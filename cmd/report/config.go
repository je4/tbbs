package main

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"path/filepath"
	"strings"
	"time"
)

type Endpoint struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type Forward struct {
	Local  Endpoint `toml:"local"`
	Remote Endpoint `toml:"remote"`
}

type SSHTunnel struct {
	User       string             `toml:"user"`
	PrivateKey string             `toml:"privatekey"`
	Endpoint   Endpoint           `toml:"endpoint"`
	Forward    map[string]Forward `toml:"forward"`
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type DBMySQL struct {
	DSN            string
	ConnMaxTimeout duration
	Schema         string
}

// main config structure for toml file
type Config struct {
	CertPem     string               `toml:"certpem"`
	KeyPem      string               `toml:"keypem"`
	LogFile     string               `toml:"logfile"`
	LogLevel    string               `toml:"loglevel"`
	LogFormat   string               `toml:"logformat"`
	AccessLog   string               `toml:"accesslog"`
	BaseDir     string               `toml:"basedir"`
	Addr        string               `toml:"addr"`
	AddrExt     string               `toml:"addrext"`
	User        string               `toml:"user"`
	Password    string               `toml:"password"`
	Tunnel      map[string]SSHTunnel `toml:"tunnel"`
	DB          DBMySQL              `toml:"db"`
	TemplateDir string               `toml:"templatedir"`
	StaticDir   string               `toml:"staticdir"`
	Logo        string               `toml:"logo"`
	Institution string               `toml:"institution"`
}

func LoadConfig(fp string, conf *Config) error {
	_, err := toml.DecodeFile(fp, conf)
	if err != nil {
		return errors.Wrapf(err, "error loading config file %v", fp)
	}
	conf.BaseDir = strings.TrimRight(filepath.ToSlash(conf.BaseDir), "/")
	return nil
}
