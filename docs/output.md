目前输出内容使用json格式。
#### MySQL协议的解析结果示例如下：
```
{"cip":"192.XX.XX.1","cport":63888,"sip":"192.XX.XX.2","sport":3306,"user":"root","db":"sniffer","sql":"show tables","cpr":1.0,"bt":1566545734147,"cms":15}
```
其中cip代表客户端ip，cport代表客户端port(客户端ip：port组成session标识)，sip代表server ip，sport代表server port，user代表查询用户，db代表当前连接的库名，sql代表查询语句，cpr代表抓包率，bt代表查询开始时间戳，cms代表查询消耗的时间，单位是毫秒
