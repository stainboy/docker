// Package fifo provides logdriver emits log messages into fifo
package fifo

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
)

// Name is the name of the file that the fifo logs to.
const Name = "fifo"

// fifolog is Logger implementation for default Docker logging.
type fifolog struct {
	appenders []logger.Logger
}

func init() {
	if err := logger.RegisterLogDriver(Name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(Name, ValidateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

// New creates new fifolog which writes to filename passed in
// on given context.
func New(ctx logger.Context) (logger.Logger, error) {
	l := &fifolog{
		appenders: []logger.Logger{},
	}

	for _, def := range fifoOpts.defs {
		if a, err := createAppender(ctx, &fifoOpts, &def); err != nil {
			return nil, err
		} else {
			l.appenders = append(l.appenders, a)
		}
	}
	return l, nil
}

// Log converts logger.Message to jsonlog.JSONLog and serializes it to file.
func (l *fifolog) Log(msg *logger.Message) error {
	for _, a := range l.appenders {
		a.Log(msg)
	}
	return nil
}

// Close closes underlying file and signals all readers to stop.
func (l *fifolog) Close() error {
	for _, a := range l.appenders {
		a.Close()
	}
	return nil
}

// Name returns name of this logger.
func (l *fifolog) Name() string {
	return Name
}
