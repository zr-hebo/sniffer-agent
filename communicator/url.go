package communicator

func init()  {
	router.Path("/check_alive").Methods("GET").HandlerFunc(outletCheckAlive)
	router.Path("/get_config").Methods("GET").HandlerFunc(outletGetConfig)
	router.Path("/set_config").Methods("POST").HandlerFunc(outletSetConfig)
}