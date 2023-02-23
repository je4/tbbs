package main

import (
	"database/sql"
	_ "github.com/dgraph-io/badger/v3"
	_ "github.com/go-sql-driver/mysql"
	"github.com/je4/bagarc/v2/pkg/bagit"
	"github.com/je4/tbbs/v2/pkg/tbbs"
	"github.com/je4/utils/v2/pkg/ssh"
	flag "github.com/spf13/pflag"
	"log"
	"time"
)

func main() {
	var action = flag.String("action", "bagit", "ingest")
	var basedir = flag.String("basedir", ".", "base folder with archived bagit's")
	var configfile = flag.String("cfg", "/etc/tbbs.toml", "configuration file")
	var tempdir = flag.String("temp", "/tmp", "folder for temporary files")

	flag.Parse()

	var conf = &BagitConfig{
		Logfile:   "",
		Loglevel:  "DEBUG",
		Logformat: `%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} > %{level:.5s} - %{message}`,
		Checksum:  []string{"md5", "sha512"},
		Tempdir:   "/tmp",
	}
	if err := LoadBagitConfig(*configfile, conf); err != nil {
		log.Printf("cannot load config file: %v", err)
	}

	// set all config values, which could be orverridden by flags
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "temp":
			conf.Tempdir = *tempdir
		case "basedir":
			conf.BaseDir = *basedir
		}
	})

	logger, lf := bagit.CreateLogger("bagit", conf.Logfile, nil, conf.Loglevel, conf.Logformat)
	defer lf.Close()

	var tunnels []*ssh.SSHtunnel
	for name, tunnel := range conf.Tunnel {
		logger.Infof("starting tunnel %s", name)

		forwards := make(map[string]*ssh.SourceDestination)
		for fwName, fw := range tunnel.Forward {
			forwards[fwName] = &ssh.SourceDestination{
				Local: &ssh.Endpoint{
					Host: fw.Local.Host,
					Port: fw.Local.Port,
				},
				Remote: &ssh.Endpoint{
					Host: fw.Remote.Host,
					Port: fw.Remote.Port,
				},
			}
		}

		t, err := ssh.NewSSHTunnel(
			tunnel.User,
			tunnel.PrivateKey,
			&ssh.Endpoint{
				Host: tunnel.Endpoint.Host,
				Port: tunnel.Endpoint.Port,
			},
			forwards,
			logger,
		)
		if err != nil {
			logger.Errorf("cannot create tunnel %v@%v:%v - %v", tunnel.User, tunnel.Endpoint.Host, tunnel.Endpoint.Port, err)
			return
		}
		if err := t.Start(); err != nil {
			logger.Errorf("cannot create configfile %v - %v", t.String(), err)
			return
		}
		tunnels = append(tunnels, t)
	}
	defer func() {
		for _, t := range tunnels {
			t.Close()
		}
	}()
	// if tunnels are made, wait until connection is established
	if len(conf.Tunnel) > 0 {
		time.Sleep(2 * time.Second)
	}

	var db *sql.DB
	var err error
	if conf.DB.DSN != "" {
		logger.Debugf("connecting mysql database")
		db, err = sql.Open("mysql", conf.DB.DSN)
		if err != nil {
			// don't write dsn in error message due to password inside
			logger.Panicf("error connecting to database: %v", err)
			return
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			logger.Panicf("cannot ping database: %v", err)
			return
		}
		db.SetConnMaxLifetime(time.Duration(conf.DB.ConnMaxTimeout.Duration))
	}

	switch *action {
	case "ingest":
		i, err := tbbs.NewIngest(conf.Tempdir, conf.KeyDir, conf.IngestLocation, conf.Reportdir, db, conf.DB.Schema, conf.PrivateKey, logger)
		if err != nil {
			logger.Fatalf("cannot create BagitIngest: %v", err)
			return
		}
		if err := i.Ingest(); err != nil {
			logger.Fatalf("cannot ingest: %v", err)
			return
		}
		if err := i.Transfer(); err != nil {
			logger.Fatalf("cannot ingest: %v", err)
			return
		}
		if err := i.Check(); err != nil {
			logger.Fatalf("cannot ingest: %v", err)
			return
		}
		/*
			if err := i.Report(); err != nil {
				logger.Fatalf("cannot create report: %v", err)
				return
			}
		*/
	case "report":
		i, err := tbbs.NewIngest(conf.Tempdir, conf.KeyDir, conf.IngestLocation, conf.Reportdir, db, conf.DB.Schema, conf.PrivateKey, logger)
		if err != nil {
			logger.Fatalf("cannot create BagitIngest: %v", err)
			return
		}
		if err := i.Report(); err != nil {
			logger.Fatalf("cannot create report: %v", err)
			return
		}
	default:
		logger.Errorf("invalid action: %s", *action)
	}

}
