package loghub

import (
	"time"
	"github.com/Sirupsen/logrus"

	"github.com/docker/docker/daemon/logger"
	circuit "github.com/rubyist/circuitbreaker"
)

type secureLogger struct {
	logger.Logger
	*circuit.Breaker
	timeout time.Duration
}

func newSecureLogger(l logger.Logger, c int64, t int) logger.Logger {
	// Stop future calls if c consecutive failure occurs
	return &secureLogger{l,
		circuit.NewConsecutiveBreaker(c),
		time.Microsecond * time.Duration(t)}
}

func (sl *secureLogger) Log(m *logger.Message) error {
	// Threat 1s timeout as failure

	// Avoid using built-in Call(fn, timeout) which is performance penalty
	// return l.Breaker.Call(func() error {
	// 	return l.Logger.Log(m)
	// }, time.Second)

	cb := sl.Breaker
	if cb.Ready() {
		// Breaker is not tripped, proceed

		begin := time.Now()
		err := sl.Logger.Log(m)
		elapse := time.Since(begin)

		if elapse > sl.timeout {
			cb.Fail()
			logrus.Debugf("Time out found in circuit breaker: %v / %v", sl.timeout, elapse)
		} else if err != nil {
			cb.Fail()
			logrus.Debug(err)
		} else {
			cb.Success()
		}

		return nil
	}
	logrus.Warn("Log hub circuit break is open")

	// Breaker is in a tripped state.
	return circuit.ErrBreakerOpen
}
