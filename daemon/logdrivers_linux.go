package daemon

import (
	// Importing packages here only to make sure their init gets called and
	// therefore they register themselves to the logdriver factory.
	_ "github.com/docker/docker/daemon/logger/jsonfilelog"
	// _ "github.com/docker/docker/daemon/logger/kafkalog"
	_ "github.com/docker/docker/daemon/logger/loghub"
	_ "github.com/docker/docker/daemon/logger/fifo"
	// _ "github.com/docker/docker/daemon/logger/redislog"
)
