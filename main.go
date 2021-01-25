package main

import (
	"flag"
	"fmt"
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
	"github.com/zr-hebo/sniffer-agent/session-dealer/mysql"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/zr-hebo/sniffer-agent/capture"
	"github.com/zr-hebo/sniffer-agent/communicator"
	"github.com/zr-hebo/sniffer-agent/exporter"
)

var (
	logLevel string
)

func init()  {
	flag.StringVar(&logLevel, "log_level", "warn", "log level. Default is info")
}

func initLog()  {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	switch logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		panic(fmt.Sprintf("cannot set log level:%s, there have four types can set: debug, info, warn, error", logLevel))
	}
}

func main()  {
	flag.Parse()
	prepareEnv()

	go communicator.Server()
	mainServer()
}

func mainServer()  {
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

func prepareEnv()  {
	initLog()
	sd.CheckParams()
	mysql.PrepareEnv()
}