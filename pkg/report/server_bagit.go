package report

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func (s *Server) bagitHandler(w http.ResponseWriter, req *http.Request) {
	if s.dev {
		if err := s.InitTemplates(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-type", "text/plain")
			w.Write([]byte(fmt.Sprintf("cannot initialize templates: %v", err)))
			return
		}
	}
	vars := mux.Vars(req)
	bagitID, err := strconv.ParseInt(vars["bagitid"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("bagitid %s not a number: %v", vars["bagitid"], err)))
		return
	}

	overview, err := s.stats.BagitOverview(bagitID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("error querying statistics: %v", err)))
		return
	}
	tpl := s.templates["bagit.gohtml"]
	if err := tpl.Execute(w, overview); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("error executing template %s : %v", "overview.gohtml", err)))
		return
	}
}
