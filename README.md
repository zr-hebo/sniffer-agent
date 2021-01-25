# Sniffer-Agent

> Sniffer TCP package, parsed with mysql protocol, optional you can just print on screen or send query info to Kafka.
> 抓取tcp包解析出mysql语句，将查询信息打印在屏幕上或者发送到Kafka

### 1. Architecture

架构设计：

本项目采用模块化设计，主要分为四大模块：TCP抓包模块，协议解析模块，输出模块，心跳模块
![架构设计图](https://github.com/zr-hebo/sniffer-agent/blob/master/images/arch.png)

### 2. Parse Protocol

sniffer-agent采用模块化结构，支持用户添加自己的解析模块，只要实现了统一的接口即可
- [x] MySQL
- [ ] PostgreSQL
- [ ] Redis
- [ ] Mongodb
- [ ] GRPC


##### 详细输出格式[查看](https://github.com/zr-hebo/sniffer-agent/blob/master/docs/output.md)

### 3. [CapturePacketRate](https://github.com/zr-hebo/sniffer-agent/blob/master/docs/capture_rate.md)
sniffer-agent可以动态设置抓包率，详情[查看文档](https://github.com/zr-hebo/sniffer-agent/blob/master/docs/capture_rate.md)

### 4. Exporter

输出模块主要负责，将解析的结果对外输出。默认情况下输出到命令行，可以通过指定export_type参数选择kafka，这时候会直接将解析结果发送到kafka。
同样只要实现了export接口，用户可以自定义自己的输出方式。

### 5. Install:

环境：

golang：1.12+

libpcap包

测试脚本运行在python3环境下


1.安装依赖，目前自测支持Linux系列操作系统，其他版本的系统有待验证

CentOS:
```
yum install libpcap-devel
```

Ubuntu:
```
apt-get install libpcap-dev
```
2.执行编译命令 go build

### 6. Demo

目前只支持MySQL协议的抓取，需要将编译后的二进制文件上传到MySQL服务器上

1.最简单的使用

`./sniffer-agent`

2.指定log级别，可以指定的值为debug、info、warn、error，默认是info

`./sniffer-agent --log_level=debug`

默认会监听 网卡：eth0，端口3306

3.指定网卡和监听端口

`./sniffer-agent --interface=eth0 --port=3358`

4.指定输出到kafka，为了将ddl和select、dml区分处理，这里使用了两个topic来生产消息

`./sniffer-agent --export_type=kafka --kafka-server=$kafka_server:$kafka_server --kafka-group-id=sniffer --kafka-async-topic=non_ddl_sql_collector --kafka-sync-topic=ddl_sql_collector`

5.指定严格模式，通过查询获取长连接的用户名和数据库

`./sniffer-agent --strict_mode=true --admin_user=root --admin_passwd=123456`

#### 7. 题外话
在做这个功能之前，项目组调研过类似功能的产品，最有名的是 [mysql-sniffer](https://github.com/Qihoo360/mysql-sniffer) 和 [go-sniffer](https://github.com/40t/go-sniffer)，这两个产品都很优秀，不过我们的业务场景要求更多。
我们需要将提取的SQL信息发送到kafka进行处理，之前的两个产品输出的结果需要进行一些处理然后自己发送，在QPS比较高的情况下，这些处理会消耗较多的CPU；
另外mysql-sniffer使用c++开发，平台的适用性较差，后期扩展较难。
开发的过程中也借鉴了这些产品的思想，另外在MySQL包解析的时候，参考了一些 [Vitess](https://github.com/vitessio/vitess) 和 [TiDB](https://github.com/pingcap/tidb) 的内容，部分私有变量和函数直接复制使用，这里向这些优秀的产品致敬，如有侵权请随时联系。

#### 8. 结果分析
在压测的过程中和mysql-sniffer进行了结果对比，压测执行28万条语句，mysql-sniffer抓取了8千条，sniffer-agent抓取了30万条语句（其中包含client自动生成的语句）

#### 9. 风险提示
1.sniffer-agent使用了pacp抓包，根据pacp抓包原理，在IO较高的时候有一定的概率丢包；

2.sniffer-agent提供了Prepare语句的支持，但是如果sniffer-agent在prepare语句初始化之后启动，就无法抓取prepare语句；

3.目前在 MySQL5.5-5.7上测试可用，MySQL8上会出现一些莫名其妙的问题；

4.目前为止也没有使用 go mod进行包管理，因为一些原因，依赖的一些包在国内没法直接下载进来，因此把这些包保存在 vendor目录，方便编译；

##### License [MIT](https://opensource.org/licenses/MIT)

