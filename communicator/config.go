package communicator

import (
	"flag"
	"net/http"
	"time"

	_ "net/http/pprof"
	"github.com/gorilla/mux"
)

var (
	communicatePort int
)

func init()  {
	flag.IntVar(&communicatePort, "communicate_port", 8088, "http server port. Default is 8088")
}

func Server()  {
	server := &http.Server{
		Addr:        ":" + string(communicatePort),
		Handler:     mux.NewRouter(),
		IdleTimeout: time.Second * 5,
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

