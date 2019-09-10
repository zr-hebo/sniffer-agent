package communicator

import (
	"flag"
	"github.com/gorilla/mux"
	_ "net/http/pprof"
	"sync"
)

const (
	CAPTURE_PACKET_RATE = "capture_packet_rate"
)

var (
	communicatePort int
	// capturePacketRate float64
	router = mux.NewRouter()
)

var (
	configMapLock     sync.RWMutex
	configMap         map[string]configItem
	catpurePacketRate *capturePacketRateConfig
)

func init() {
	catpurePacketRate = newCapturePacketRateConfig()

	flag.IntVar(&communicatePort, "communicate_port", 8088, "http server port. Default is 8088")
	var cpr float64
	flag.Float64Var(&cpr, CAPTURE_PACKET_RATE, 1, "capture packet rate. Default is 1.0")
	_ = catpurePacketRate.setVal(cpr)

	configMap = make(map[string]configItem)
}
