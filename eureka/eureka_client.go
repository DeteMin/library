package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/HikoQiu/go-eureka-client/eureka"
)

const defaultPort = 80

type EurekaCfg struct {
	Eureka Eureka `yaml:"eureka" json:"eureka"`
}

type Eureka struct {
	ServerName     string `yaml:"server_name" json:"server_name"`
	EurekaZone     string `yaml:"eureka_zone" json:"eureka_zone"`
	Port           int    `yaml:"port" json:"port"`
	HealthInterval int    `yaml:"health_interval" json:"health_interval"`
}

func UnmarshalConfig(fileName string) *EurekaCfg {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("read eureka config err %v", err)
	}
	var config EurekaCfg
	if err := yaml.Unmarshal([]byte(data), &config); err != nil { //解析yaml文件
		log.Fatalf("read eureka config err %v", err)
	}
	return &config
}

type options struct {
	ip                string
	port              int
	logFunc           LogFunc
	headers           map[string]string
	keepAliveDuration time.Duration
}

// Option 配置
type Option interface {
	apply(*options)
}

// LogFunc 日志方法
type LogFunc = func(level int, format string, a ...interface{})

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

// WithPort 指定端口，默认80
func WithPort(port int) Option {
	return optionFunc(func(o *options) {
		o.port = port
	})
}

// WithIP 指定IP，默认当前主机IP
func WithIP(ip string) Option {
	return optionFunc(func(o *options) {
		o.ip = ip
	})
}

// WithLogFunc 指定logger
/* 例子
func(level int, format string, a ...interface{}) {
	var logFunc *zerolog.Event
	switch level {
	case 1:
		logFunc = log.Debug()
	case 2:
		logFunc = log.Info()
	case 3:
		logFunc = log.Error()
	}
	if logFunc != nil {
		funcName, file, line, _ := runtime.Caller(2)
		fullFuncName := runtime.FuncForPC(funcName).Name()
		arr := strings.Split(fullFuncName, "/")
		arrFile := strings.Split(file, "/")

		logFunc.Str("file", arrFile[len(arrFile)-1]).Int("line", line).Str("func", arr[len(arr)-1]).Msgf(format, a...)
	}
}
*/
func WithLogFunc(logFunc LogFunc) Option {
	return optionFunc(func(o *options) {
		o.logFunc = logFunc
	})
}

// WithHeaders 指定请求头 默认{"Content-Type": "application/json"}
func WithHeaders(headers map[string]string) Option {
	return optionFunc(func(o *options) {
		for k, v := range headers {
			o.headers[k] = v
		}
	})
}

// WithKeepAliveDuration 实例健康检测时常，默认1分钟
func WithKeepAliveDuration(keepAliveDuration time.Duration) Option {
	return optionFunc(func(o *options) {
		o.keepAliveDuration = keepAliveDuration
	})
}

// Client 整合客户端
type Client interface {
	Register() error
}
type discoveryClient struct {
	eurekaCli *eureka.Client
	log       LogFunc
}

// New 新建客户端
/*
	serviceName: 服务名，如：post-service
	zone: eureka服务地址，多个逗号","分隔离，如：http://192.168.1.100:1111/eureka,http://192.168.1.101:1111/eureka
	opts: 可选配置，指定端口：eureka.WithPort(80)，指定IP：eureka.WithIP("192.168.1.102"), 指定logger：eureka.WithLogFunc(func...)
	调用例子：
		client1 := eureka.New("post-service", "http://192.168.1.100:1111/eureka,http://192.168.1.101:1111/eureka");
		client2 := eureka.New(
			"post-service",
			"http://192.168.1.100:1111/eureka",
			eureka.WithPort(80),
			eureka.WithIP("192.168.1.102"),
			eureka.WithLogFunc(func...)
		);
*/
func New(serviceName string, zone string, opts ...Option) Client {
	options := options{
		port:              defaultPort,
		keepAliveDuration: time.Minute,
		logFunc: func(level int, format string, a ...interface{}) {
			switch level {
			case eureka.LevelDebug:
				format = "[debug] " + format
			case eureka.LevelInfo:
				format = "[info] " + format
			case eureka.LevelError:
				format = "[error] " + format
			}
			funcName, file, line, _ := runtime.Caller(2)
			fullFuncName := runtime.FuncForPC(funcName).Name()
			arr := strings.Split(fullFuncName, "/")
			arrFile := strings.Split(file, "/")

			log.Printf(fmt.Sprintf("%s %s:%d ", arr[len(arr)-1], arrFile[len(arrFile)-1], line)+format, a...)
		},
	}

	for _, o := range opts {
		o.apply(&options)
	}

	if options.logFunc != nil {
		eureka.SetLogger(options.logFunc)
	}

	eurekaConfig := eureka.GetDefaultEurekaClientConfig()
	eurekaConfig.UseDnsForFetchingServiceUrls = false
	eurekaConfig.ServiceUrl = map[string]string{
		eureka.DEFAULT_ZONE: zone,
	}
	eurekaClient := new(eureka.Client).Config(eurekaConfig).Register(serviceName, options.port)

	if "" != options.ip {
		vo := eurekaClient.GetInstance()
		vo.Hostname = options.ip
		vo.IppAddr = options.ip
		eurekaClient = eurekaClient.RegisterVo(vo)
	}

	eurekaClient.Run()

	cli := discoveryClient{
		eurekaCli: eurekaClient,
		log:       options.logFunc,
	}
	go cli.keepAlive(options.keepAliveDuration)
	return &cli
}

func (cli *discoveryClient) keepAlive(duration time.Duration) {
	for {
		api, err := cli.eurekaCli.Api()
		if err != nil {
			cli.log(eureka.LevelError, "%v", err)
			time.Sleep(duration)
			continue
		}

		instance := cli.eurekaCli.GetInstance()
		vo, err := api.QuerySpecificAppInstance(instance.InstanceId)
		if vo == nil || err != nil {
			cli.log(eureka.LevelError, "%v", err)
			if err := cli.Register(); err != nil {
				cli.log(eureka.LevelError, "%v", err)
			}
			time.Sleep(duration)
			continue
		}
		if vo.Status != eureka.STATUS_UP {
			if err := api.UpdateInstanceStatus(instance.App, instance.InstanceId, eureka.STATUS_UP); err != nil {
				cli.log(eureka.LevelError, "%v", err)
			}
			time.Sleep(duration)
			continue
		}
		time.Sleep(duration)
	}
}

// Register 重新注册实例
func (cli *discoveryClient) Register() error {
	api, err := cli.eurekaCli.Api()
	if err != nil {
		return err
	}
	instance := cli.eurekaCli.GetInstance()
	instanceID, err := api.RegisterInstanceWithVo(instance)
	if err != nil {
		return err
	}
	instance.InstanceId = instanceID
	cli.eurekaCli.RegisterVo(instance)
	err = api.UpdateInstanceStatus(instance.App, instance.InstanceId, eureka.STATUS_UP)
	if err != nil {
		return err
	}

	return nil
}
