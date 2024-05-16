package eureka

import "testing"

func Test_Register(t *testing.T) {
	eClient := New("GW-LABEL-EXTRACT-QR", "http://hddata:hddata$2019@192.168.1.59:8002/eureka", WithPort(8015), WithIP("192.168.1.59"))
	eClient.Register()

	select {}
}

func Test_GetApplicationByServerName(t *testing.T) {
	eClient := New("GW-LABEL-EXTRACT-QR", "http://hddata:hddata$2019@192.168.1.218:8002/eureka", WithPort(8025), WithIP("192.168.1.59"))
	eClient.Register()

	//获取eureka上注册的服务
	app := eClient.GetApplicationByServerName("GW-CACHE-SERVICE")

	t.Logf("%v", app.Instances[0])
	t.Logf("app ip:%v port:%v", app.Instances[0].IppAddr, app.Instances[0].Port)

	select {}
}
