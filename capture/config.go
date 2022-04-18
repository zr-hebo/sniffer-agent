package capture

import (
	"math/rand"
	"time"

	log "github.com/golang/glog"
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
)

var (
	localIPAddr *string

	sessionPool = make(map[string]sd.ConnSession)
	// sessionPoolLock sync.Mutex
)

func init() {
	ipAddr, err := getLocalIPAddr()
	if err != nil {
		panic(err)
	}

	localIPAddr = &ipAddr

	rand.Seed(time.Now().UnixNano())
}

func ShowLocalIP() {
	log.Infof("parsed local ip address:%s", *localIPAddr)
}
