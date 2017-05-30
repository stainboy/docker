package fifo

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/docker/docker/daemon/logger"
)

type fifoMetaService struct {
	def  *fifoDef
	meta io.WriteCloser
}

func createMetaService(def *fifoDef) (*fifoMetaService, error) {

	// mkdir -p
	rootDir := def.dir
	if err := os.MkdirAll(rootDir, os.ModeDir); err != nil {
		return nil, err
	}

	// rm -f .meta
	name := fmt.Sprintf("%s/.meta", rootDir)
	if err := os.RemoveAll(name); err != nil {
		return nil, err
	}

	// mkfifo .meta
	meta, err := mkfifo(name, false)
	if err != nil {
		return nil, err
	}

	m := &fifoMetaService{def, meta}
	return m, nil
}

func (m *fifoMetaService) PushEvent(ctx logger.Context) error {

	if ev, err := render(ctx, m.def.meta, 1); err != nil {
		return err
	} else {
		data := []byte(ev + "\n")
		if _, err := m.meta.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func render(ctx logger.Context, tpl string, source int) (string, error) {

	funcMap := template.FuncMap{
		"labels": func(key string) string {
			return ctx.ContainerLabels[key]
		},
		"source": func() string {
			return strconv.Itoa(source)
		},
		"bash": func(text string) string {
			cmd := exec.Command("bash", "-c", text)
			var buff bytes.Buffer
			cmd.Stdout = &buff
			cmd.Stderr = &buff
			cmd.Run()
			return buff.String()
		},
		"inet": func(iface string) string {
			if ifaces, err := net.Interfaces(); err == nil {
				for _, i := range ifaces {
					if strings.Contains(iface, i.Name) {
						if addrs, err := i.Addrs(); err == nil {
							for _, addr := range addrs {
								var ip net.IP
								switch v := addr.(type) {
								case *net.IPNet:
									ip = v.IP
								case *net.IPAddr:
									ip = v.IP
								}
								return ip.String()
							}
						}
					}
				}
			}
			return "127.0.0.1"
		},
	}

	// compile template
	t, err := template.New("").Funcs(funcMap).Parse(tpl)
	if err != nil {
		return "", err
	}

	// interpret
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func renderBoolean(ctx logger.Context, tpl string) (bool, error) {
	s, err := render(ctx, tpl, 1)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(s)
}
