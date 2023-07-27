package cache

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

var (
	CacheCLient = NewCache(&redis.Options{
		Addr:     "192.168.1.83:6379",
		Password: "hddata",
		DB:       0,
	})
)

type RSP struct {
	List string
}

func TestCache_Get(t *testing.T) {
	var rsp = make([]*RSP, 0)
	f := func() (interface{}, error) {
		return &[]*RSP{
			{
				List: "Hello world",
			},
		}, nil
	}
	key := "gw:" + fmt.Sprintf("TestCache_Get:%s", "1")
	err := CacheCLient.Get(context.Background(), key, &rsp, 3*time.Minute, f)
	if err != nil {
		t.Errorf("cache get err:%v", err)
		t.Fail()
		return
	}
	for i, r := range rsp {
		t.Logf("cache get index:%d value:%v", i, r.List)
	}

}
