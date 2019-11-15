package communicator

import (
	"fmt"
	"math"
)

type configItem interface {
	setVal (interface{}) error
	getVal () interface{}
}

type capturePacketRateConfig struct {
	name     string
	tcpCPR   float64
	mysqlCPR float64
}

func newCapturePacketRateConfig() (cprc *capturePacketRateConfig) {
	cprc = &capturePacketRateConfig{
		name:     CAPTURE_PACKET_RATE,
		tcpCPR:   1.0,
		mysqlCPR: 1.0,
	}
	return
}

func (cprc *capturePacketRateConfig) setVal (val interface{}) (err error){
	realVal, ok := val.(float64)
	if !ok {
		err = fmt.Errorf("cannot reansform val: %v to float64", val)
		return
	}

	fmt.Printf("set config %s: %v\n", CAPTURE_PACKET_RATE, realVal)
	cprc.mysqlCPR = realVal
	cprc.tcpCPR = math.Sqrt(realVal)
	return
}

func (cprc *capturePacketRateConfig) getVal () (val interface{}){
	return cprc.mysqlCPR
}