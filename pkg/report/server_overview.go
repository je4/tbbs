package report

import (
	"fmt"
	"net/http"
)

func (s *Server) overviewHandler(w http.ResponseWriter, req *http.Request) {
	if s.dev {
		if err := s.InitTemplates(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-type", "text/plain")
			w.Write([]byte(fmt.Sprintf("cannot initialize templates: %v", err)))
			return
		}
	}
	overview, err := s.stats.Overview()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("error querying statistics: %v", err)))
		return
	}
	tpl := s.templates["overview.gohtml"]
	if err := tpl.Execute(w, struct {
		Institution, Logo string
		Overview          *Overview
	}{
		Institution: s.institution,
		Logo:        s.logo,
		Overview:    overview,
	}); err != nil {

		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("error executing template %s : %v", "overview.gohtml", err)))
		return
	}
}
