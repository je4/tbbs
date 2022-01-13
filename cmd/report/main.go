package main

import (
	"context"
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/je4/tbbs/v2/pkg/report"
	lm "github.com/je4/utils/v2/pkg/logger"
	"github.com/je4/utils/v2/pkg/ssh"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var err error

	var basedir = flag.String("basedir", ".", "base folder with html contents")
	var configfile = flag.String("cfg", "/etc/report.toml", "configuration file")
	var dev = flag.Bool("dev", false, "reload templates on every access")
	flag.Parse()

	var config = &Config{
		LogFile:   "",
		LogLevel:  "DEBUG",
		LogFormat: `%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} [%{shortfile}] > %{level:.5s} - %{message}`,
		BaseDir:   *basedir,
		Addr:      "localhost:80",
		AddrExt:   "http://localhost:80/",
		User:      "jane",
		Password:  "doe",
	}
	if err := LoadConfig(*configfile, config); err != nil {
		log.Printf("cannot load config file: %v", err)
	}

	// create logger instance
	logger, lf := lm.CreateLogger("Salon Digital", config.LogFile, nil, config.LogLevel, config.LogFormat)
	defer lf.Close()

	var accessLog io.Writer
	var f *os.File
	if config.AccessLog == "" {
		accessLog = os.Stdout
	} else {
		f, err = os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logger.Panicf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accessLog = f
	}

	var tunnels []*ssh.SSHtunnel
	for name, tunnel := range config.Tunnel {
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
	if len(config.Tunnel) > 0 {
		time.Sleep(2 * time.Second)
	}

	var db *sql.DB
	logger.Debugf("connecting mysql database")
	db, err = sql.Open("mysql", config.DB.DSN)
	if err != nil {
		// don't write dsn in error message due to password inside
		logger.Panicf("error connecting to database: %v", err)
		return
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		logger.Panicf("cannot ping database: %v", err)
		return
	}
	db.SetConnMaxLifetime(config.DB.ConnMaxTimeout.Duration)

	stats, err := report.NewStatistics(db, "tbbs", logger)
	if err != nil {
		logger.Panicf("cannot create statistics module: %v", err)
		return
	}

	var staticFS fs.FS
	if config.StaticDir != "" {
		staticFS = os.DirFS(config.StaticDir)
	} else {
		staticFS, err = fs.Sub(report.StaticFS, "static")
		if err != nil {
			logger.Panicf("cannot get subtree of embedded static: %v", err)
			return
		}
	}

	var templateFS fs.FS
	if config.TemplateDir != "" {
		templateFS = os.DirFS(config.TemplateDir)
	} else {
		templateFS, err = fs.Sub(report.TemplateFS, "template")
		if err != nil {
			logger.Panicf("cannot get subtree of embedded template: %v", err)
			return
		}
	}

	srv, err := report.NewServer(
		"TBBS",
		config.Addr,
		config.AddrExt,
		config.User,
		config.Password,
		logger,
		accessLog,
		stats,
		staticFS,
		templateFS,
		*dev,
	)
	if err != nil {
		logger.Panicf("cannot initialize server: %v", err)
	}
	go func() {
		if err := srv.ListenAndServe(config.CertPem, config.KeyPem); err != nil {
			log.Fatalf("server died: %v", err)
		}
	}()

	end := make(chan bool, 1)

	// process waiting for interrupt signal (TERM or KILL)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)

		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		<-sigint

		// We received an interrupt signal, shut down.
		logger.Infof("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	logger.Info("server stopped")

}
