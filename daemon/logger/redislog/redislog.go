// Package redislog provides logdriver emits log messages into redis
package redislog

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
	"github.wdf.sap.corp/sigma-anywhere/saptail"
)

// Name is the name of the file that the redis logs to.
const Name = "redis"

var tail saptail.Interface

// redislog is Logger implementation for default Docker logging.
type redislog struct {
	writer saptail.Interface
	extra  logger.LogAttributes
}

func init() {
	if err := logger.RegisterLogDriver(Name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(Name, ValidateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

// New creates new redislog which writes to filename passed in
// on given context.
func New(ctx logger.Context) (logger.Logger, error) {
	extra := extractLabels(ctx.Config, ctx.ContainerLabels)
	log := redislog{
		writer: tail,
		extra:  extra,
	}
	return &log, nil
}

// ValidateLogOpt looks for log options `server` (e.g. redis.smec.sap.corp:6379)
func ValidateLogOpt(cfg map[string]string) error {
	for key := range cfg {
		switch key {
		case "server":
		case "port":
		case "database":
		default:
			if strings.Index(key, "labels/") != 0 {
				return fmt.Errorf("unknown log opt '%s' for redis log driver", key)
			}
		}
	}
	return validateAndInit(cfg["server"], cfg["port"], cfg["database"])
}

// Log converts logger.Message to jsonlog.JSONLog and serializes it to file.
func (l *redislog) Log(msg *logger.Message) error {
	m := logger.Message{
		Line:      msg.Line,
		Source:    msg.Source,
		Timestamp: msg.Timestamp,
		Attrs:     l.extra,
		// Partial:   msg.Partial,
	}
	return l.writer.Message(&m, false)
}

// Close closes underlying file and signals all readers to stop.
func (l *redislog) Close() error {
	return nil
}

// Name returns name of this logger.
func (l *redislog) Name() string {
	return Name
}

func validateAndInit(server, port, database string) error {
	if tail != nil {
		return nil
	}

	logrus.Infof("Initialize redis connection to %s:%s.%s", server, port, database)

	db, err := strconv.Atoi(database)
	if err != nil {
		return err
	}
	tail, err = saptail.New(server, port, db)
	if err != nil {
		return err
	}

	return nil
}

func extractLabels(config map[string]string, labels map[string]string) logger.LogAttributes {
	extra := logger.LogAttributes{}

	funcMap := template.FuncMap{
		"labels": func(key string) string {
			return labels[key]
		},
	}

	for k, v := range config {
		if strings.Index(k, "labels/") == 0 {
			if tokens := strings.SplitN(k, "/", 2); len(tokens) == 2 {
				key := tokens[1]
				if t, err := template.New("").Funcs(funcMap).Parse(v); err != nil {
					logrus.Error(err)
					extra[key] = v
				} else {
					buf := &bytes.Buffer{}
					if err := t.Execute(buf, labels); err != nil {
						logrus.Error(err)
						extra[key] = v
					} else {
						extra[key] = buf.String()
					}
				}
			}
		}
	}

	return extra
}
