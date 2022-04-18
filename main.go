package main

import (
	"flag"
	"os"

	log "github.com/golang/glog"
	"github.com/zr-hebo/sniffer-agent/capture"
	"github.com/zr-hebo/sniffer-agent/communicator"
	"github.com/zr-hebo/sniffer-agent/exporter"
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
	"github.com/zr-hebo/sniffer-agent/session-dealer/mysql"
)

var (
	logLevel string
)

func init() {
	flag.StringVar(&logLevel, "log_level", "warn", "log level. Default is info")
}

func initLog() {
}

func main() {
	flag.Parse()
	prepareEnv()

	go communicator.Server()
	mainServer()
}

func mainServer() {
	ept := exporter.NewExporter()
	networkCard := capture.NewNetworkCard()
	log.Info("begin listen")
	for queryPiece := range networkCard.Listen() {
		err := ept.Export(queryPiece)
		if err != nil {
			log.Error(err.Error())
		}
		queryPiece.Recovery()
	}

	log.Errorf("cannot get network package from %s", capture.DeviceName)
	os.Exit(1)
}

func prepareEnv() {
	initLog()
	sd.CheckParams()
	mysql.PrepareEnv()
	capture.ShowLocalIP()
}
