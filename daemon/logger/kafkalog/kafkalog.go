package kafkalog

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
	kafka "gopkg.in/Shopify/sarama.v1"
)

//Name of logger type
const Name = "kafka"

var loggerFactory *factory

var supportedOpts = map[string]string{
	"formatted-logkey":   "",
	"disired-containers": "",
	"brokers":            "",
	"topic":              "",
	"required-ack":       "",
	"compression":        "",
	"flush/frequency":    "",
	"flush/bytes":        "",
	"max-message-bytes":  ""}

type kafkaLogger struct {
	messageKey string
	topic      string
	valid      bool
	sinker     kafka.AsyncProducer
}

func init() {
	if err := logger.RegisterLogDriver(Name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(Name, ValidateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

func (l *kafkaLogger) Log(msg *logger.Message) error {
	if !l.valid {
		return nil
	}

	m := logEntry{
		encoded: msg.Line,
	}

	l.sinker.Input() <- &kafka.ProducerMessage{
		Topic: l.topic,
		Key:   kafka.StringEncoder(l.messageKey),
		Value: &m,
	}

	return nil
}

func (l *kafkaLogger) Name() string {
	return Name
}

// Close closes underlying file and signals all readers to stop.
func (l *kafkaLogger) Close() error {
	return nil
}

// String generates descritpion
func (l *kafkaLogger) String() string {
	return fmt.Sprintf("message Key: %v, topic: %v, valid: %v", l.messageKey, l.topic, l.valid)
}

//ValidateLogOpt do a check and initialize singleton kafka producer
func ValidateLogOpt(cfg map[string]string) error {
	logrus.Info("validate logger opt")
	for key := range cfg {
		if _, ok := supportedOpts[key]; !ok {
			return fmt.Errorf("unknown log opt '%s' for kafka log driver", key)
		}
	}

	return initFactory(cfg)
}

func initFactory(cfg map[string]string) error {
	if loggerFactory != nil {
		return nil
	}

	brokers, ok := cfg["brokers"]
	if !ok {
		return fmt.Errorf("brokers are needed")
	}
	brokerList := strings.Split(brokers, ",")
	logrus.Debugf("Kafka-logger# broker list: %v", brokerList)

	topic, ok := cfg["topic"]
	if !ok {
		return fmt.Errorf("topic is needed")
	}
	logrus.Debugf("Kafka-logger# topic: %v", topic)

	formatKey, ok := cfg["formatted-logkey"]
	if !ok {
		return fmt.Errorf("formatted-logkey is needed")
	}
	logrus.Debugf("Kafka-logger# formatted-logkey: %v", formatKey)

	config := kafka.NewConfig()
	if val, ok := cfg["required-ack"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		config.Producer.RequiredAcks = kafka.RequiredAcks(v)
		logrus.Debugf("Kafka-logger# RequiredAcks: %v", v)
	}

	if val, ok := cfg["compression"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		config.Producer.Compression = kafka.CompressionCodec(v)
		logrus.Debugf("Kafka-logger# Compression: %v", v)
	}

	if val, ok := cfg["flush/frequency"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		config.Producer.Flush.Frequency = time.Duration(v) * time.Millisecond
		logrus.Debugf("Kafka-logger# flush/frequency: %v milli seconds", v)
	}

	if val, ok := cfg["flush/bytes"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		config.Producer.Flush.Bytes = v
		logrus.Debugf("Kafka-logger# flush/bytes: %v", v)
	}

	if val, ok := cfg["max-message-bytes"]; ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		config.Producer.MaxMessageBytes = v
		logrus.Debugf("Kafka-logger# max-message-bytes: %v", v)
	}

	sinker, err := kafka.NewAsyncProducer(brokerList, config)
	if err != nil {
		return fmt.Errorf("failed to start Kafka producer: %v", err)
	}

	loggerFactory = &factory{
		sinker:    sinker,
		topic:     topic,
		formatKey: formatKey,
	}

	//It is concatenated with ","
	if val, ok := cfg["disired-containers"]; ok {
		loggerFactory.disiredContainers = strings.Split(val, ",")
		logrus.Debugf("Kafka-logger# disired-containers: %v", val)
	}

	go func() {
		for err := range sinker.Errors() {
			logrus.Error("Failed to write log entry: %v", err)
		}
	}()

	return nil
}

//New creates new redislog which writes to filename passed in
//on given context.
func New(ctx logger.Context) (logger.Logger, error) {
	return loggerFactory.Create(ctx)
}

type logEntry struct {
	encoded []byte
}

func (le *logEntry) Length() int {
	return len(le.encoded)
}

func (le *logEntry) Encode() ([]byte, error) {
	return le.encoded, nil
}

type factory struct {
	sinker            kafka.AsyncProducer
	topic             string
	formatKey         string
	disiredContainers []string
}

func (f *factory) Create(ctx logger.Context) (logger.Logger, error) {
	funcMap := template.FuncMap{
		"labels": func(key string) string {
			return ctx.ContainerLabels[key]
		},
	}

	l := &kafkaLogger{}
	if t, err := template.New("").Funcs(funcMap).Parse(f.formatKey); err != nil {
		logrus.Warnf("Initial kafaka driver failed %v", err)
	} else {
		buf := &bytes.Buffer{}
		if err := t.Execute(buf, ctx.ContainerLabels); err != nil {
			logrus.Warnf("Initial kafaka driver failed %v", err)
		} else {
			l.messageKey = buf.String()
			l.valid = true
		}
	}

	l.sinker = f.sinker
	l.topic = f.topic

	if f.disiredContainers != nil {
		l.valid = false
		v, ok := ctx.ContainerLabels["io.kubernetes.container.name"]
		logrus.Infof("check current container name %v", v)
		if ok {
			for _, cn := range f.disiredContainers {
				if cn == v {
					l.valid = true
					break
				}
			}
		}
	}

	logrus.Infof("Kafka-logger# New kafka logger is created: %v", l)

	return l, nil
}
