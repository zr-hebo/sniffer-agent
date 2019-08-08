package exporter

import (
	"flag"

	"github.com/zr-hebo/sniffer-agent/model"
)

var (
	exportType string
)

func init() {
	flag.StringVar(&exportType,"export_type", "cli", "export type. Default is cli, that is command line")
}

type Exporter interface {
	Export(model.QueryPiece) error
}

func NewExporter() Exporter {
	switch exportType {
	case "cli":
		return NewCliExporter()
	case "kafka":
		return NewKafkaExporter()
	default:
		return NewCliExporter()
	}
}
