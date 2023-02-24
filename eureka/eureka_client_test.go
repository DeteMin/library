package utils

import "testing"

func Test_Register(t *testing.T) {
	eClient := New("GW-LABEL-EXTRACT-QR", "http://hddata:hddata$2019@192.168.1.59:8002/eureka", WithPort(8015), WithIP("192.168.1.59"))
	eClient.Register()

	select {}
}
