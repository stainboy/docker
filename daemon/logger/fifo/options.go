package fifo

import (
	"fmt"
	"strings"
	"sync"

	"strconv"

	"github.com/Sirupsen/logrus"
	units "github.com/docker/go-units"
)

type fifoOptions struct {
	bufferSize int64
	defs       []fifoDef
}

type fifoDef struct {
	name   string
	dir    string
	logkey string
	filter string
	meta   string
	mixstd bool

	service *fifoMetaService
}

var (
	fifoOpts fifoOptions
	once     sync.Once
)

// ValidateLogOpt initializes fifo options
func ValidateLogOpt(cfg map[string]string) error {
	var err error
	once.Do(func() {
		err = loadDef(cfg)
	})
	return err
}

func loadDef(cfg map[string]string) error {

	var err error
	fifoOpts.bufferSize, err = units.FromHumanSize(cfg["buffer-size"])
	if err != nil {
		return err
	}

	fifoOpts.defs = make([]fifoDef, 0)
	for k, v := range cfg {
		if tokens := strings.SplitN(k, "/", 2); len(tokens) == 2 {
			scope := tokens[0]
			field := tokens[1]
			def := findOrInitDef(&fifoOpts, scope)
			switch field {
			case "dir":
				def.dir = v
			case "logkey":
				def.logkey = v
			case "filter":
				def.filter = v
			case "meta":
				def.meta = v
			case "mixstd":
				if def.mixstd, err = strconv.ParseBool(v); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown log opt '%s' for fifo log driver", k)
			}
		}
	}

	for _, d := range fifoOpts.defs {
		if d.service, err = createMetaService(&d); err != nil {
			return err
		}
		logrus.Infof("Successfully loaded fifo def %s", d.name)
	}

	logrus.Infof("Successfully loaded fifo logs driver")
	return nil
}

func findOrInitDef(opts *fifoOptions, name string) *fifoDef {
	for _, d := range opts.defs {
		if d.name == name {
			return &d
		}
	}
	d := fifoDef{
		name: name,
	}
	opts.defs = append(opts.defs, d)
	return &d
}
