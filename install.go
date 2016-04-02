package main

import (
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/kardianos/osext"
)

const serviceFile = `
[Unit]
Description=LED Lighting Controller

[Service]
ExecStart={{.BinPath}} run -c {{ .ConfigFile }}

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

	binPath := filepath.Join(prefix, "usr/bin/lablights2")
	os.MkdirAll(filepath.Dir(binPath), 0755)
	dst, err := os.OpenFile(binPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
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

	dstPath := filepath.Join(prefix, "usr/lib/systemd/system/lablights2.service")
	os.MkdirAll(filepath.Dir(dstPath), 0755)
	dst, err = os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()

	err = serviceTmpl.Execute(dst, struct{ BinPath, ConfigFile string }{binPath, configPath})
	if err != nil {
		return err
	}
	dst.Close()

	dstPath = filepath.Join(prefix, configPath)
	_, err = os.Stat(dstPath)
	if err == nil && !reset {
		return nil
	}

	confFlags := os.O_CREATE | os.O_WRONLY
	if reset {
		confFlags |= os.O_TRUNC
	}

	os.MkdirAll(filepath.Dir(dstPath), 0755)
	dst, err = os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.WriteString(dst, configFile)
	if err != nil {
		return err
	}
	dst.Close()

	return nil
}
