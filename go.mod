module gitlab.hddata.cn/common/library

go 1.19

require (
	github.com/HikoQiu/go-eureka-client/eureka v0.0.0-20200428035747-ac92e3f91f92
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/HikoQiu/go-eureka-client/eureka v0.0.0-20200428035747-ac92e3f91f92 => github.com/DeteMin/go-eureka-client/eureka v0.0.0-20221212030601-56bec0537f55

require (
	github.com/miekg/dns v1.0.15 // indirect
	golang.org/x/crypto v0.3.0 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	gopkg.in/resty.v1 v1.10.2 // indirect
)
