package capture

import (
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
	log "github.com/sirupsen/logrus"
)

var (
	localIPAddr *string

	sessionPool = make(map[string]sd.ConnSession)
)

func init() {
	ipAddr, err := getLocalIPAddr()
	if err != nil {
		panic(err)
	}

	localIPAddr = &ipAddr
	log.Infof("parsed local ip address:%s", *localIPAddr)
}
