package kafkalog

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
	kafka "gopkg.in/Shopify/sarama.v1"
)

//Name of logger type
const Name = "kafka"

var loggerFactory *factory

var manager *kafkaManager

type kafkaLogger struct {
	messageKey string
	topic      string
	valid      bool
	sinker     kafkaSinker
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

	err := l.sinker.Log(&kafka.ProducerMessage{
		Topic: l.topic,
		Key:   kafka.StringEncoder(l.messageKey),
		Value: &m,
	})

	if err != nil {
		err = fmt.Errorf("Kafka sent failure: %v", string(msg.Line))
	}

	return err
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
	var err error
	manager, err = newManager(cfg)

	if err == nil {
		manager.Start()
	}

	return err
}

//New creates new redislog which writes to filename passed in
//on given context.
func New(ctx logger.Context) (logger.Logger, error) {
	funcMap := template.FuncMap{
		"labels": func(key string) string {
			return ctx.ContainerLabels[key]
		},
	}

	l := &kafkaLogger{}
	if t, err := template.New("").Funcs(funcMap).Parse(manager.formatKey); err != nil {
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

	l.sinker = manager
	l.topic = manager.topic

	if manager.disiredContainers != nil {
		l.valid = false
		v, ok := ctx.ContainerLabels["io.kubernetes.container.name"]
		logrus.Infof("check current container name %v", v)
		if ok {
			for _, cn := range manager.disiredContainers {
				if cn == v {
					l.valid = true
					break
				}
			}
		}
	}

	logrus.Infof("New kafka logger is created: %v", l)

	return l, nil
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
