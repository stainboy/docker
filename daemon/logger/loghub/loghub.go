// Package loghub provides a special log driver which enables multiple log drivers being used at the same time
package loghub

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
)

const Name = "loghub"

var config loghubConfig

type loghubConfig struct {
	Drivers []struct {
		Name    string            `json:"name"`
		Enabled bool              `json:"enabled"`
		Options map[string]string `json:"options"`
	} `json:"drivers"`
}

type loghubLogger struct {
	reader  logger.LogReader
	writers []logger.Logger
}

func init() {
	if err := logger.RegisterLogDriver(Name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(Name, ValidateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

// New creates new loghubLogger which writes to filename passed in
// on given context.
func New(ctx logger.Context) (logger.Logger, error) {

	hub := loghubLogger{
		writers: []logger.Logger{},
	}

	for _, driver := range config.Drivers {
		if driver.Enabled {
			creator, err := logger.GetLogDriver(driver.Name)
			if err != nil {
				return nil, err
			}
			cc, err := makeContext(ctx, driver.Name, driver.Options)
			if err != nil {
				return nil, err
			}
			writer, err := creator(cc)
			if err != nil {
				return nil, err
			}
			hub.writers = append(hub.writers, writer)

			reader, ok := writer.(logger.LogReader)
			if ok {
				hub.reader = reader
			}
		}
	}

	return &hub, nil
}

// ValidateLogOpt looks for log options `config`.
func ValidateLogOpt(cfg map[string]string) error {
	for key := range cfg {
		switch key {
		case "config":
		default:
			return fmt.Errorf("unknown log opt '%s' for loghub log driver", key)
		}
	}
	return validateAndInit(cfg["config"])
}

// Log converts logger.Message to jsonlog.JSONLog and serializes it to file.
func (l *loghubLogger) Log(msg *logger.Message) error {
	for _, writer := range l.writers {
		writer.Log(msg)
	}
	return nil
}

// Close closes underlying file and signals all readers to stop.
func (l *loghubLogger) Close() error {
	for _, writer := range l.writers {
		writer.Close()
	}
	return nil
}

// Name returns name of this logger.
func (l *loghubLogger) Name() string {
	return Name
}

func validateAndInit(filename string) error {
	if len(config.Drivers) != 0 {
		return nil
	}

	logrus.Infof("Loghub driver is going to load configuration from %s", filename)

	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, &config)
	if err != nil {
		return err
	}

	for _, driver := range config.Drivers {
		if driver.Enabled {
			err = logger.ValidateLogOpts(driver.Name, driver.Options)
			if err != nil {
				return err
			}
			logrus.Infof("Loghub driver has successfully validated sub driver %s", driver.Name)
		}
	}

	logrus.Infoln("Loghub driver was fully loaded and configured")
	return nil
}

func makeContext(ctx logger.Context, name string, cfg map[string]string) (logger.Context, error) {
	ctx.Config = cfg
	return ctx, nil
}
