package kafkalog

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	kafka "gopkg.in/Shopify/sarama.v1"
	"github.com/rubyist/circuitbreaker"
)

var logCounter = 0
var supportedOpts = map[string]string{
	"formatted-logkey":   "",
	"desired-containers": "",
	"brokers":            "",
	"topic":              "",
	"required-ack":       "",
	"compression":        "",
	"flush/frequency":    "",
	"flush/bytes":        "",
	"producer/retry":     "",
	"max-message-bytes":  "",
	"logger-channel-size":"",
	"circuit-breaker-consec-count":"",
	"circuit-breaker-timeout":""}

type kafkaSinker interface {
	//This is none blocking function
	Log(*kafka.ProducerMessage) error
}

//kafkamanager
type kafkaManager struct {
	msgBuf chan *kafka.ProducerMessage

	brokerList        []string
	topic             string
	formatKey         string
	desiredContainers []string
	config            *kafka.Config
	breaker      	  *circuit.Breaker
	breakerTimeout    time.Duration
	ready	          uint32


}

//newManager is to create a kafka manager with provided config. The parsed configuration will be used to build real kafka producer laterly
func newManager(cfg map[string]string) (*kafkaManager, error) {
	for key := range cfg {
		if _, ok := supportedOpts[key]; !ok {
			return nil, fmt.Errorf("unknown log opt '%s' for kafka log driver", key)
		}
	}

	channelSize := 10000
	if val, ok := cfg["logger-channel-size"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		channelSize = v
	}

	bcc := 3
	if val, ok := cfg["circuit-breaker-consec-count"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		bcc = v
	}

	//Microseconds
	breakerTimeout := 100
	if val, ok := cfg["circuit-breaker-timeout"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		breakerTimeout = v
	}

 	km := &kafkaManager{
		msgBuf: make(chan *kafka.ProducerMessage, channelSize),
		breakerTimeout: time.Microsecond * time.Duration(breakerTimeout),
		breaker: circuit.NewConsecutiveBreaker(int64(bcc)),
	}

	brokers, ok := cfg["brokers"]
	if !ok {
		return nil, fmt.Errorf("brokers are needed")
	}
	km.brokerList = strings.Split(brokers, ",")
	logrus.Debugf("Kafka-logger# broker list: %v", km.brokerList)

	km.topic, ok = cfg["topic"]
	if !ok {
		return nil, fmt.Errorf("topic is needed")
	}
	logrus.Debugf("Kafka-logger# topic: %v", km.topic)

	km.formatKey, ok = cfg["formatted-logkey"]
	if !ok {
		return nil, fmt.Errorf("formatted-logkey is needed")
	}
	logrus.Debugf("Kafka-logger# formatted-logkey: %v", km.formatKey)

	//It is concatenated with ","
	if val, ok := cfg["desired-containers"]; ok {
		km.desiredContainers = strings.Split(val, ",")
		logrus.Debugf("Kafka-logger# desired-containers: %v", val)
	}

	km.config = kafka.NewConfig()
	if val, ok := cfg["required-ack"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		km.config.Producer.RequiredAcks = kafka.RequiredAcks(v)
		logrus.Debugf("Kafka-logger# RequiredAcks: %v", v)
	}

	if val, ok := cfg["compression"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		km.config.Producer.Compression = kafka.CompressionCodec(v)
		logrus.Debugf("Kafka-logger# Compression: %v", v)
	}

	if val, ok := cfg["flush/frequency"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		km.config.Producer.Flush.Frequency = time.Duration(v) * time.Millisecond
		logrus.Debugf("Kafka-logger# flush/frequency: %v milli seconds", v)
	}

	if val, ok := cfg["flush/bytes"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		km.config.Producer.Flush.Bytes = v
		logrus.Debugf("Kafka-logger# flush/bytes: %v", v)
	}

	if val, ok := cfg["max-message-bytes"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		km.config.Producer.MaxMessageBytes = v
		logrus.Debugf("Kafka-logger# max-message-bytes: %v", v)
	}

	if val, ok := cfg["producer/retry"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		km.config.Producer.Retry.Max = v
		logrus.Debugf("Kafka-logger# producer/retry: %v", v)
	}

	km.config.Version = kafka.V0_10_1_0
	km.config.Producer.Return.Successes = true
	km.config.ClientID = "Kafka-logger"

	return km, nil
}

func (km *kafkaManager) Log(msg *kafka.ProducerMessage) error {
	if atomic.LoadUint32(&km.ready) == 0 {
		return fmt.Errorf("Kafka producer is not ready, fail to send log[Key:%v] to kafka", msg.Key)
	}

	if km.breaker.Ready(){
		begin := time.Now()
		km.msgBuf <- msg
		elapse := time.Since(begin)
		if elapse > km.breakerTimeout {
			km.breaker.Fail()
		}
	} else {
		logCounter += 1
		if logCounter == 1000{
			logrus.Infof("Circuit breaker is open for Kafka logger")
			logCounter = 0
		}
	}

	return nil
}

func (km *kafkaManager) Start() {
	go km.start()
}

func (km *kafkaManager) start() {
	toSleep := time.Duration(1) * time.Second
	producer, err := kafka.NewAsyncProducer(km.brokerList, km.config)
	for err != nil {
		logrus.Warnf("Can't initialize Kafka producer: %v", err)
		logrus.Warnf("Try to retry to create Kafka producer after %v second", toSleep)
		time.Sleep(toSleep)
		if toSleep <= 128*time.Second {
			toSleep *= time.Duration(2)
		}
		producer, err = kafka.NewAsyncProducer(km.brokerList, km.config)
	}

	go func(p kafka.AsyncProducer, m <-chan *kafka.ProducerMessage) {
		logrus.Debugf("Kafka Manager begins to checkout incomming messages and drop them into the producer")
		for msg := range m {
			p.Input() <- msg
		}
	}(producer, km.msgBuf)

	go func(p kafka.AsyncProducer) {
 		for err := range p.Errors() {
			km.breaker.Fail()
			//if kafka connection failed, each message will produce a message including "circuit breaker is open", which is too many
			if !strings.Contains(err.Error(), "circuit") {
				logrus.Warnf("Failed to write log entry: %v", err)
			}
		}
	}(producer)

	go func(p kafka.AsyncProducer) {
		for range p.Successes() {
			km.breaker.Success()
		}
	}(producer)
	//Finally set a flag
	atomic.StoreUint32(&km.ready, 1)
}
