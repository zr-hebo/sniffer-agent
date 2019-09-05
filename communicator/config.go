package communicator

import (
	"flag"
	"github.com/gorilla/mux"
	_ "net/http/pprof"
	"sync"
)

const (
	THROW_PACKET_RATE = "throw_packet_rate"
)

var (
	communicatePort int
	router = mux.NewRouter()
)

var (
	configMapLock sync.RWMutex
	configMap map[string]configItem
)

func init()  {
	flag.IntVar(&communicatePort, "communicate_port", 8088, "http server port. Default is 8088")

	configMap = make(map[string]configItem)
	tprc := newThrowPacketRateConfig()
	configMap[tprc.name] = tprc
}
