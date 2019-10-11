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
	router = mux.NewRouter()
)

var (
	configMapLock     sync.RWMutex
	configMap         map[string]configItem
	catpurePacketRate *capturePacketRateConfig
	catpurePacketRateVal float64
)

func init() {
	catpurePacketRate = newCapturePacketRateConfig()

	flag.IntVar(&communicatePort, "communicate_port", 8088, "http server port. Default is 8088")
	flag.Float64Var(&catpurePacketRateVal, CAPTURE_PACKET_RATE, 1.0, "capture packet rate. Default is 1.0")

	configMap = make(map[string]configItem)
}
