package loghub

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
)

func Test_initAndValidate(t *testing.T) {

	var config loghubConfig
	fmt.Print(config)
	fmt.Print(config.Drivers)
}

func Test_extractLabels(t *testing.T) {
	// config := map[string]string{
	// 	"labels/node":      "192.168.1.22",
	// 	"labels/namespace": "$(io.kubernetes.pod.namespace)",
	// 	"labels/pod":       "$(io.kubernetes.pod.name)",
	// }
	config := map[string]string{
		"labels/node":   "192.168.1.22",
		"labels/logkey": "10.58.113.105:qq:{{labels \"io.kubernetes.pod.namespace\"}}:{{labels \"io.kubernetes.pod.name\"}}:{{labels \"io.kubernetes.container.name\"}}",
	}
	labels := map[string]string{
		"io.kubernetes.pod.namespace":  "cy",
		"io.kubernetes.pod.name":       "bss-234766563-df311",
		"io.kubernetes.container.name": "nginx",
	}

	attr := extractLabels(config, labels)
	fmt.Print(attr)
}

func Test_validateLabels(t *testing.T) {
	key := "labels/landscape"
	if strings.Index(key, "labels/") != 0 {
		fmt.Printf("unknown log opt '%s' for redis log driver", key)
	} else {
		fmt.Println("ok!")
	}
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
