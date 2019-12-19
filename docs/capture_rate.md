## CapturePacketRate

通过API可以动态获取或者设置抓包率，基于此项功能，sniffer提供了动态调整抓包率率的功能，比如在QPS低的时候设置抓包率为1，在QPS高的时候设置为0.01

#### Get CapturePacketRate
```
curl 'http://127.0.0.1:8088/get_config?config_name=capture_packet_rate'
```

#### Set CapturePacketRate
```
curl -XPOST -d'{"config_name":"capture_packet_rate","value":0.8}' 'http://127.0.0.1:8088/set_config?config_name=capture_packet_rate'
```

