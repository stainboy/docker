package loghub

import (
	"github.com/docker/docker/daemon/logger"
)

func (l *loghubLogger) ReadLogs(config logger.ReadConfig) *logger.LogWatcher {
	if l.reader == nil {
		return nil
	}
	return l.reader.ReadLogs(config)
}
