package communicator

import (
	"fmt"
	"math"
)

type configItem interface {
	setVal (interface{}) error
	getVal () interface{}
}

type throwPacketRateConfig struct {
	name string
	tcpTPR float64
	mysqlTPR float64
}

func newThrowPacketRateConfig() (tpr *throwPacketRateConfig) {
	tpr = &throwPacketRateConfig{
		name: THROW_PACKET_RATE,
		tcpTPR: 0.0,
		mysqlTPR: 0.0,
	}
	return
}

func (tc *throwPacketRateConfig) setVal (val interface{}) (err error){
	realVal, ok := val.(float64)
	if !ok {
		err = fmt.Errorf("cannot reansform val: %v to float64", val)
		return
	}

	tc.mysqlTPR = realVal
	tc.tcpTPR = math.Sqrt(realVal)
	return
}

func (tc *throwPacketRateConfig) getVal () (val interface{}){
	return tc.mysqlTPR
}