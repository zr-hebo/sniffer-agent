package capture

import (
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
	log "github.com/sirupsen/logrus"
)

var (
	localIPAddr string

	sessionPool = make(map[string]sd.ConnSession)
)

func init() {
	var err error
	localIPAddr, err = getLocalIPAddr()
	if err != nil {
		panic(err)
	}

	log.Infof("parsed local ip address:%s", localIPAddr)
}
