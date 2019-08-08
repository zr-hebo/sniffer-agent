package exporter

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/Shopify/sarama"
	"github.com/zr-hebo/sniffer-agent/model"
)

var (
	ddlPatern = regexp.MustCompile(`(?i)^\s*(create|alter|drop)`)
	kafkaServer string
	kafkaGroupID string
	asyncTopic string
	syncTopic string
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
}

type kafkaExporter struct {
	asyncProducer sarama.AsyncProducer
	syncProducer  sarama.SyncProducer
	asyncTopic  string
	syncTopic  string
}

func checkParams()  {
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
	if ddlPatern.MatchString(qp.GetSQL()) {
		log.Debugf("deal ddl: %s\n", qp.String())

		msg := &sarama.ProducerMessage{
			Topic: ke.syncTopic,
			Value: sarama.StringEncoder(qp.String()),
		}
		_, _, err = ke.syncProducer.SendMessage(msg)
		if err != nil {
			return
		}

	} else {
		log.Debugf("deal non ddl: %s", qp.String())
		msg := &sarama.ProducerMessage{
			Topic: ke.asyncTopic,
			Value: sarama.ByteEncoder(qp.Bytes()),
		}

		ke.asyncProducer.Input() <- msg
	}
	return
}
