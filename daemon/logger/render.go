package logger

import (
	"bytes"
	"net"
	"os/exec"
	"strings"
	"text/template"
)

func Render(ctx Context, tpl string) (string, error) {

	funcMap := template.FuncMap{
		"labels": func(key string) string {
			return ctx.ContainerLabels[key]
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
