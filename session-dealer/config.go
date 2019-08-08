package session_dealer

import (
	"flag"
)

const (
	ServiceTypeMysql = "mysql"
)

var (
	serviceType string
)

func init() {
	flag.StringVar(&serviceType, "service_type", "mysql", "service type. Default is mysql")
}

