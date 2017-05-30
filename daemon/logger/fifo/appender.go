package fifo

import (
	"io"
	"io/ioutil"

	"github.com/docker/docker/daemon/logger"
)

type fifoAppender struct {
	logger.Logger
	stdout io.WriteCloser
	stderr io.WriteCloser
}

func createAppender(ctx logger.Context, opts *fifoOptions, def *fifoDef) (logger.Logger, error) {

	var app fifoAppender

	// test match filter
	if filter, err := renderBoolean(ctx, def.filter); err != nil {
		return nil, err
	} else if !filter {
		app.stdout = NopWriterCloser(ioutil.Discard)
		app.stderr = app.stdout
		return &app, nil
	}

	// mkfifo for stdout
	if logkey, err := render(ctx, def.logkey, 1); err != nil {
		return nil, err
	} else if app.stdout, err = mkfifo(logkey, true); err != nil {
		return nil, err
	}

	// mkfifo for stderr
	if def.mixstd {
		app.stderr = app.stdout
	} else if logkey, err := render(ctx, def.logkey, 2); err != nil {
		return nil, err
	} else if app.stderr, err = mkfifo(logkey, true); err != nil {
		return nil, err
	}

	// push meta change event
	if err := def.service.PushEvent(ctx); err != nil {
		return nil, err
	}

	return &app, nil
}

// Log converts logger.Message to jsonlog.JSONLog and serializes it to file.
func (l *fifoAppender) Log(msg *logger.Message) error {
	if msg.Source == "stdout" {
		l.stdout.Write(msg.Line)
	} else {
		l.stderr.Write(msg.Line)
	}
	return nil
}

// Close closes underlying file and signals all readers to stop.
func (l *fifoAppender) Close() error {
	l.stdout.Close()
	l.stderr.Close()
	return nil
}

// Name returns name of this logger.
func (l *fifoAppender) Name() string {
	return ""
}
