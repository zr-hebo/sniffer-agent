package communicator

import "fmt"

type configItem interface {
	setVal (interface{}) error
	getVal () interface{}
}

type throwPacketRateConfig struct {
	name string
	value float64
}

func newThrowPacketRateConfig() (tpr *throwPacketRateConfig) {
	tpr = &throwPacketRateConfig{
		name: THROW_PACKET_RATE,
		value: 0.0,
	}
	return
}

func (tc *throwPacketRateConfig) setVal (val interface{}) (err error){
	realVal, ok := val.(float64)
	if !ok {
		err = fmt.Errorf("cannot reansform val: %v to float64", val)
		return
	}

	tc.value = realVal
	return
}

func (tc *throwPacketRateConfig) getVal () (val interface{}){
	return tc.value
}
