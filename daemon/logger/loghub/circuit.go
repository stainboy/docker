package loghub

import (
	"time"

	"github.com/docker/docker/daemon/logger"
	circuit "github.com/rubyist/circuitbreaker"
)

type secureLogger struct {
	logger.Logger
	*circuit.Breaker
}

func newSecureLogger(l logger.Logger) logger.Logger {
	// Stop future calls if 3 consecutive failure occurs
	return &secureLogger{l, circuit.NewConsecutiveBreaker(3)}
}

func (l *secureLogger) Log(m *logger.Message) error {
	// Threat 1s timeout as failure

	// Avoid using built-in Call(fn, timeout) which is performance penalty
	// return l.Breaker.Call(func() error {
	// 	return l.Logger.Log(m)
	// }, time.Second)

	cb := l.Breaker
	if cb.Ready() {
		// Breaker is not tripped, proceed

		begin := time.Now()
		err := l.Logger.Log(m)
		since := time.Since(begin)

		if since > 1*time.Second {
			cb.Fail()
			return circuit.ErrBreakerTimeout
		}
		if err != nil {
			cb.Fail()
		}
		cb.Success()
	}
	// Breaker is in a tripped state.
	return circuit.ErrBreakerOpen
}
