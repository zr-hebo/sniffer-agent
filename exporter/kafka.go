package exporter

import (
	"flag"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	log "github.com/golang/glog"
	"github.com/zr-hebo/sniffer-agent/model"
)

var (
	kafkaServer string
	kafkaGroupID string
	asyncTopic string
	syncTopic string
	compress string
    compressType sarama.CompressionCodec
)

func init() {
	flag.StringVar(
	 	&kafkaServer, "kafka-server", "", "kafka server address. No default value")
	flag.StringVar(
	 	&kafkaGroupID,
		"kafka-group-id", "", "kafka service group. No default value")
	flag.StringVar(
	 	&asyncTopic,
		"kafka-async-topic", "", "kafka async send topic. No default value")
	flag.StringVar(
		&syncTopic,
		"kafka-sync-topic", "", "kafka sync send topic. No default value")
	flag.StringVar(
		&compress,
		"compress-type", "", "kafka message compress type. Default value is no compress")
}

type kafkaExporter struct {
	asyncProducer sarama.AsyncProducer
	syncProducer  sarama.SyncProducer
	asyncTopic  string
	syncTopic  string
}

func checkParams()  {
	switch compress {
	case "":
		compressType = sarama.CompressionNone
	case "gzip":
		compressType = sarama.CompressionGZIP
	case "snappy":
		compressType = sarama.CompressionSnappy
	case "lz4":
		compressType = sarama.CompressionLZ4
	default:
		panic(fmt.Sprintf("cannot support kafka compress type: %s", compress))
	}

	fmt.Printf("kafka message compress type: %s", compress)
	params := make(map[string]string)
	params["kafka-server"] = kafkaServer
	params["kafka-group-id"] = kafkaGroupID
	params["kafka-async-topic"] = asyncTopic
	params["kafka-sync-topic"] = syncTopic
	for param := range params {
		if len(params[param]) < 1{
			panic(fmt.Sprintf("%s cannot be empty", param))
		}
	}
}

func NewKafkaExporter() (ke *kafkaExporter) {
	checkParams()
	ke = &kafkaExporter{}
	conf := sarama.NewConfig()
	conf.Producer.Return.Successes = true
	conf.ClientID = kafkaGroupID
	conf.Producer.Compression = compressType
	addrs := strings.Split(kafkaServer, ",")
	syncProducer, err := sarama.NewSyncProducer(addrs, conf)
	if err != nil {
		panic(err.Error())
	}
	ke.syncProducer = syncProducer

	asyncProducer, err := sarama.NewAsyncProducer(addrs, conf)
	if err != nil {
		panic(err.Error())
	}
	ke.asyncProducer = asyncProducer
	ke.asyncTopic = asyncTopic
	ke.syncTopic = syncTopic

	go func() {
		errors := ke.asyncProducer.Errors()
		success := ke.asyncProducer.Successes()
		for {
			select {
			case err := <-errors:
				if err != nil {
					log.Error(err.Error())
				}

			case <-success:
			}
		}
	}()
	return
}

func (ke *kafkaExporter) Export (qp model.QueryPiece) (err error){
	defer func() {
		if err != nil {
			log.Errorf("export with kafka failed <-- %s", err.Error())
		}
	}()

	if qp.NeedSyncSend() {
		// log.Debugf("deal ddl: %s\n", *qp.String())

		msg := &sarama.ProducerMessage {
			Topic: ke.syncTopic,
			Value: sarama.ByteEncoder(qp.Bytes()),
		}
		_, _, err = ke.syncProducer.SendMessage(msg)
		if err != nil {
			return
		}

	} else {
		// log.Debugf("deal non ddl: %s", *qp.String())
		msg := &sarama.ProducerMessage {
			Topic: ke.asyncTopic,
			Value: sarama.ByteEncoder(qp.Bytes()),
		}

		ke.asyncProducer.Input() <- msg
	}

	return
}
