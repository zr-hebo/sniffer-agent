package communicator

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	hu "github.com/zr-hebo/util-http"
	"github.com/zr-hebo/validator"
)

func Server() {
	initConfig()

	server := &http.Server{
		Addr:        "0.0.0.0:" + strconv.Itoa(communicatePort),
		IdleTimeout: time.Second * 5,
	}

	http.Handle("/", router)
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func initConfig()  {
	_ = catpurePacketRate.setVal(catpurePacketRateVal)
}

func outletCheckAlive(resp http.ResponseWriter, req *http.Request) {
	mp := hu.NewMouthpiece(resp)
	defer func() {
		_ = mp.Convey()
	}()

	mp.Data = "OK"
}

func outletGetConfig(resp http.ResponseWriter, req *http.Request) {
	mp := hu.NewMouthpiece(resp)
	defer func() {
		_ = mp.Convey()
	}()

	ep := &struct {
		ConfigName string  `validate:"nonzero" json:"config_name"`
	}{}
	up := hu.NewUnpacker(req, ep, nil)
	if err := up.Unpack(); err != nil {
		mp.Err = err
		return
	}
	if err := validator.Validate(*ep); err != nil {
		mp.Err = err
		return
	}

	_, ok := configMap[ep.ConfigName]
	if !ok {
		mp.Err = fmt.Errorf("no config %s found", ep.ConfigName)
		return
	}

	mp.Data = GetConfig(ep.ConfigName)
}

func outletSetConfig(resp http.ResponseWriter, req *http.Request) {
	mp := hu.NewMouthpiece(resp)
	defer func() {
		_ = mp.Convey()
	}()

	ep := &struct {
		ConfigName string  `validate:"nonzero" json:"config_name"`
		Value interface{}  `json:"value"`
	}{}
	up := hu.NewUnpacker(req, ep, nil)
	if err := up.Unpack(); err != nil {
		mp.Err = err
		return
	}
	if err := validator.Validate(*ep); err != nil {
		mp.Err = err
		return
	}

	mp.Err = SetConfig(ep.ConfigName, ep.Value)
}

func GetTCPCapturePacketRate() float64 {
	return catpurePacketRate.tcpCPR
}

func GetMysqlCapturePacketRate() float64 {
	return catpurePacketRate.mysqlCPR
}