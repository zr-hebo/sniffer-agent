package communicator

import (
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	hu "github.com/zr-hebo/util-http"
	_ "net/http/pprof"
)

var (
	communicatePort int
	router = mux.NewRouter()
)

func init()  {
	flag.IntVar(&communicatePort, "communicate_port", 8088, "http server port. Default is 8088")

	router.Path("/get_status").Methods("GET").HandlerFunc(getStatus)
	router.Path("/set_config").Methods("POST").HandlerFunc(setConfig)
}

func Server()  {
	server := &http.Server{
		Addr:        "0.0.0.0:" + strconv.Itoa(communicatePort),
		IdleTimeout: time.Second * 5,
	}

	http.Handle("/", router)
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func getStatus(resp http.ResponseWriter, req *http.Request) {
	mp := hu.NewMouthpiece(resp)
	defer mp.Convey()
	mp.Data = "OK"
}

func setConfig(resp http.ResponseWriter, req *http.Request) {
	mp := hu.NewMouthpiece(resp)
	defer mp.Convey()
	mp.Data = "OK"
}

