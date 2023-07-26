package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

type Cache struct {
	Client *redis.Client
}

// NewCache NOTICE: 阿里云redis最大连接数为1w key命名规则: serviceName:modelName(只有model省略 model下子目录要保留):functionName:自定义参数
func NewCache(option *redis.Options) *Cache {
	return &Cache{Client: redis.NewClient(option)}
}

type CacheData struct {
	Key    string
	Value  interface{}
	Expire time.Duration
}

func (c *Cache) Put(ctx context.Context, data *CacheData) (err error) {
	// 未设置缓存时间 直接return
	if data.Expire <= 0 {
		return errors.New("未设置过期时间")
	}
	value, err := json.Marshal(data.Value)
	if err != nil {
		return
	}
	err = c.Client.Set(ctx, data.Key, value, data.Expire).Err()
	if err != nil {
		err = errors.Wrap(err, "redis set key err")
	}
	return
}

func (c *Cache) getL2Key(key string) string {
	return fmt.Sprintf("%s:l2", key)
}

func (c *Cache) getL2LockKey(key string) string {
	return fmt.Sprintf("%s:lock", key)
}

func (c *Cache) PutL2Cache(ctx context.Context, data *CacheData) (err error) {
	// 先设置二级缓存 双倍过期时间
	err = c.Put(ctx, &CacheData{
		Key:    c.getL2Key(data.Key),
		Value:  data.Value,
		Expire: data.Expire * 2,
	})
	if err != nil {
		return
	}
	return c.Put(ctx, data)
}

func (c *Cache) Del(ctx context.Context, keys ...string) (err error) {
	return c.Client.Del(ctx, keys...).Err()
}

func (c *Cache) Get(ctx context.Context, key string, data interface{}, expire time.Duration, f func() (interface{}, error)) (err error) {
	ok, err := c.get(ctx, key, data)
	if err != nil || ok {
		return
	}
	newData, err := f()
	if err != nil {
		err = errors.Wrap(err, "redis get mysql data error")
		return
	}
	err = setData(data, newData)
	if err != nil {
		return
	}

	err = c.Put(ctx, &CacheData{
		Key:    key,
		Value:  data,
		Expire: expire,
	})
	return
}

func (c *Cache) get(ctx context.Context, key string, data interface{}) (ok bool, err error) {
	value, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		err = errors.Wrap(err, "cache get key error")
		return
	}
	if err = json.Unmarshal(value, data); err != nil {
		err = errors.Wrap(err, "cache get key unmarshal error")
		return
	}
	return true, nil
}

func setData(data, newData interface{}) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = errors.New("类型错误")
		}
	}()
	v := reflect.ValueOf(data)
	if v.Len() == 0 {
		data = newData
	} else {
		newValue := reflect.ValueOf(data).Elem()
		if !newValue.CanSet() {
			err = errors.New("类型错误")
			return
		}
		newValue.Set(reflect.ValueOf(newData).Elem())
	}
	return
}

// GetL2CacheWithLock 获取二级缓存  获取到锁 应该更新缓存 未获取到锁且未获取到缓存 直接查数据库吧(说明不是热点数据)
func (c *Cache) GetL2CacheWithLock(ctx context.Context, key string, data interface{}, lockTime, cacheTime time.Duration, f func() (interface{}, error)) (err error) {
	// 先获取一级缓存
	ok, err := c.get(ctx, key, data)
	// 出错 或 获取到缓存
	if err != nil || ok {
		return
	}
	unlock, err := c.TryLock(ctx, c.getL2LockKey(key), lockTime)
	if err != nil {
		return
	}
	// 未获取到锁 查二级缓存
	if unlock == nil {
		err = c.Get(ctx, c.getL2Key(key), data, cacheTime, f)
		return
	}
	defer unlock()

	// 获取到锁 查数据库
	newData, err := f()
	if err != nil {
		return
	}
	err = setData(data, newData)
	if err != nil {
		return
	}
	err = c.PutL2Cache(ctx, &CacheData{
		Key:    key,
		Value:  data,
		Expire: cacheTime,
	})
	return
}

type UnlockFunc func() error

// TryLock unlock 为nil 即未获取到锁
func (c *Cache) TryLock(ctx context.Context, key string, t time.Duration) (unlock UnlockFunc, err error) {
	ok, err := c.Client.SetNX(ctx, key, "1", t).Result()
	if err != nil || !ok {
		return nil, err
	}
	return func() error {
		return c.Client.Del(ctx, key).Err()
	}, nil
}

// SCard 以下封装 为了wrap error(错误打印时打印堆栈)
func (c *Cache) SCard(ctx context.Context, key string) (count int64, err error) {
	count, err = c.Client.SCard(ctx, key).Result()
	if err != nil {
		err = errors.Wrap(err, "SCard error")
	}
	return
}

func (c *Cache) SRem(ctx context.Context, key string, members ...interface{}) (err error) {
	err = c.Client.SRem(ctx, key, members...).Err()
	if err != nil {
		err = errors.Wrap(err, "SRem error")
	}
	return
}

func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) (err error) {
	err = c.Client.SAdd(ctx, key, members...).Err()
	if err != nil {
		err = errors.Wrap(err, "SAdd error")
	}
	return
}
