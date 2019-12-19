### CapturePacketRate

Sniffer有一个强大的功能：可以动态设置抓报率。尤其是对比较线上性能敏感的系统，设置抓包率为从０－１的浮点数，按照该概率抓取数据包。这样能够根据系统负载情况，在采集覆盖情况和线上负载之间进行权衡。


默认抓包率为１，会处理所有抓取到的语句

#### Start with CapturePacketRate
```
./sniffer-agent --interface=eth0 --port=3358 --capture_packet_rate=1.0
```

通过API获取或者设置抓包率，比如在QPS低的时候设置抓包率为1，在QPS高的时候设置为0.01。
#### Get CapturePacketRate
```
curl 'http://127.0.0.1:8088/get_config?config_name=capture_packet_rate'
```

#### Set CapturePacketRate
```
curl -XPOST -d'{"config_name":"capture_packet_rate","value":0.01}' 'http://127.0.0.1:8088/set_config?config_name=capture_packet_rate'
```

