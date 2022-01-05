package main

import (
	"github.com/BurntSushi/toml"
	"github.com/goph/emperror"
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
type BagitConfig struct {
	CertChain      string               `toml:"*certchain"`
	PrivateKey     []string             `toml:"privatekey"`
	Logfile        string               `toml:"logfile"`
	Loglevel       string               `toml:"loglevel"`
	Logformat      string               `toml:"logformat"`
	Checksum       []string             `toml:"checksum"`
	Tempdir        string               `toml:"tempdir"`
	Reportdir      string               `toml:"reportdir"`
	KeyDir         string               `toml:"keydir"`
	DBFolder       string               `toml:"dbfolder"`
	BaseDir        string               `toml:"basedir"`
	Tunnel         map[string]SSHTunnel `toml:"tunnel"`
	DB             DBMySQL              `toml:"db"`
	IngestLocation string               `toml:"ingestloc"`
}

func LoadBagitConfig(fp string, conf *BagitConfig) error {
	_, err := toml.DecodeFile(fp, conf)
	if err != nil {
		return emperror.Wrapf(err, "error loading config file %v", fp)
	}
	conf.BaseDir = strings.TrimRight(filepath.ToSlash(conf.BaseDir), "/")
	return nil
}
