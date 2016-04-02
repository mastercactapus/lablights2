package main

import (
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/kardianos/osext"
)

const configPath = "/etc/lablights2.conf"
const serviceFile = `
[Unit]
Description=LED Lighting Controller

[Service]
ExecStart={{.Prefix}}/bin/lablights2 -c {{ .ConfigFile }}

[Install]
WantedBy=multi-user.target
`

var serviceTmpl = template.Must(template.New("service").Parse(serviceFile))

func install(prefix string, reset bool) error {
	if prefix == "" {
		prefix = "/"
	}
	bPath, err := osext.Executable()
	if err != nil {
		return err
	}

	src, err := os.Open(bPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(filepath.Join(prefix, "usr/bin/lablights2"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	src.Close()
	dst.Close()

	dst, err = os.OpenFile(filepath.Join(prefix, "usr/lib/systemd/system/lablights2.service"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()

	err = serviceTmpl.Execute(dst, struct{ Prefix, ConfigFile string }{prefix, configPath})
	if err != nil {
		return err
	}
	dst.Close()

	confFlags := os.O_WRONLY | os.O_CREATE
	if reset {
		confFlags |= os.O_TRUNC
	}
	dst, err = os.OpenFile(filepath.Join(prefix, "etc/lablights2.conf"), confFlags, 0644)
	if err != nil && reset {
		return err
	}

	return nil
}
