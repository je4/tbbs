package report

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/inhies/go-bytesize"
	dcert "github.com/je4/utils/v2/pkg/cert"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"html/template"
	"io"
	"io/fs"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	service           string
	host, port        string
	name, password    string
	srv               *http.Server
	linkTokenExp      time.Duration
	jwtKey            string
	jwtAlg            []string
	log               *logging.Logger
	AddrExt           string
	accessLog         io.Writer
	templates         map[string]*template.Template
	httpStaticServer  http.Handler
	staticFS          fs.FS
	stats             *Statistics
	dev               bool
	templateFS        fs.FS
	logo, institution string
}

func NewServer(service, addr, addrExt, name, password string,
	log *logging.Logger, accessLog io.Writer,
	stats *Statistics,
	staticFS, templateFS fs.FS,
	logo, institution string,
	dev bool) (*Server, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot split address %s", addr)
	}
	/*
		extUrl, err := url.Parse(addrExt)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse external address %s", addrExt)
		}
	*/

	srv := &Server{
		service:     service,
		host:        host,
		port:        port,
		AddrExt:     strings.TrimRight(addrExt, "/"),
		name:        name,
		password:    password,
		log:         log,
		accessLog:   accessLog,
		templates:   map[string]*template.Template{},
		stats:       stats,
		staticFS:    staticFS,
		templateFS:  templateFS,
		logo:        logo,
		institution: institution,
		dev:         dev,
	}
	srv.httpStaticServer = http.FileServer(http.FS(srv.staticFS))

	return srv, srv.InitTemplates()
}

func (s *Server) InitTemplates() error {
	var funcMap = sprig.FuncMap()
	funcMap["duration"] = func(d time.Duration) string {
		return d.String()
	}
	funcMap["formatInt64"] = func(value int64) string {
		return FormatInt(value, 3, '\'')
	}
	funcMap["formatInt"] = func(value int) string {
		return FormatInt(int64(value), 3, '\'')
	}

	funcMap["atof"] = func(value string) float64 {
		result, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return math.NaN()
		}
		return result
	}

	funcMap["addf"] = func(value, incr float64) float64 {
		return value + incr
	}

	funcMap["toMap"] = StructToMap

	funcMap["formatDuration"] = func(value float64) string {
		if math.IsNaN(value) {
			return "NaN"
		}
		d := int64(math.Floor(value))
		sec := d % 60
		d -= sec
		d /= 60
		min := d % 60
		d -= min
		d /= 60
		h := d

		return fmt.Sprintf("%02d:%02d:%.2f", h, min, float64(sec)+(value-math.Floor(value)))
	}

	funcMap["formatSize"] = func(value int64) string {
		var bs = bytesize.New(float64(value))
		if bs > bytesize.TB {
			return bs.Format("%.3f", "terabyte", false)
		}
		if bs > bytesize.GB {
			return bs.Format("%.3f", "gigabyte", false)
		}
		if bs > bytesize.MB {
			return bs.Format("%.3f", "megabyte", false)
		}
		if bs > bytesize.KB {
			return bs.Format("%.3f", "kilobyte", false)
		}
		return bs.Format("%.0f", "byte", false)
	}

	funcMap["abbrRight"] = func(l int, str string) string {
		if len(str) <= l {
			return str
		}
		runes := []rune(str)
		return "..." + string(runes[len(runes)-l:])
	}
	entries, err := fs.ReadDir(s.templateFS, ".")
	//entries, err := templateFS.ReadDir("template")
	if err != nil {
		return errors.Wrapf(err, "cannot read template folder %s", "template")
	}
	for _, entry := range entries {
		name := entry.Name()
		s.log.Debugf("initializing template %s", name)
		tpl, err := template.New(name).Funcs(funcMap).ParseFS(s.templateFS, name)
		if err != nil {
			return errors.Wrapf(err, "cannot parse template: %s", name)
		}
		s.templates[name] = tpl
	}
	return nil
}

func (s *Server) ListenAndServe(cert, key string) (err error) {
	router := mux.NewRouter()

	httpStaticServer := http.FileServer(http.FS(s.staticFS))

	router.PathPrefix("/static").Handler(
		http.StripPrefix("/static", httpStaticServer),
	).Methods("GET")

	router.HandleFunc("/", s.overviewHandler)
	router.HandleFunc("/bagit/{bagitid}", s.bagitHandler)
	router.HandleFunc("/bagit/{bagitid}/{contentid}", s.contentHandler)

	loggedRouter := handlers.CombinedLoggingHandler(s.accessLog, handlers.ProxyHeaders(router))
	addr := net.JoinHostPort(s.host, s.port)
	s.srv = &http.Server{
		Handler: loggedRouter,
		Addr:    addr,
	}

	if cert == "auto" || key == "auto" {
		s.log.Info("generating new certificate")
		cert, err := dcert.DefaultCertificate()
		if err != nil {
			return errors.Wrap(err, "cannot generate default certificate")
		}
		s.srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{*cert}}
		s.log.Infof("starting salon digital at %v - https://%s:%v/", s.AddrExt, s.host, s.port)
		return s.srv.ListenAndServeTLS("", "")
	} else if cert != "" && key != "" {
		s.log.Infof("starting salon digital at %v - https://%s:%v/", s.AddrExt, s.host, s.port)
		return s.srv.ListenAndServeTLS(cert, key)
	} else {
		s.log.Infof("starting salon digital at %v - http://%s:%v/", s.AddrExt, s.host, s.port)
		return s.srv.ListenAndServe()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
