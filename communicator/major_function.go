package communicator

import "fmt"

// SetConfig set config by config key(name) and value
func SetConfig(key string, val interface{}) (err error) {
	configMapLock.Lock()
	defer configMapLock.Unlock()

	config, ok := configMap[key]
	if !ok {
		err = fmt.Errorf("no config %s exist", key)
		return
	}

	err = config.setVal(val)
	return
}

// GetConfig get config value by config key(name)
func GetConfig(key string) (val interface{}) {
	configMapLock.RLock()
	defer configMapLock.RUnlock()

	config := configMap[key]
	return config.getVal()
}
