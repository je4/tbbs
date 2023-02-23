package report

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func (s *Server) contentHandler(w http.ResponseWriter, req *http.Request) {
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
	contentID, err := strconv.ParseInt(vars["contentid"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("contentid %s not a number: %v", vars["contentid"], err)))
		return
	}

	overview, err := s.stats.BagitOverview(bagitID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("error querying statistics: %v", err)))
		return
	}
	var content *Content
	for _, cnt := range overview.Content {
		if cnt.ContentID == contentID {
			content = cnt
			break
		}
	}
	if content == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("cannot find content #%v in bagit #%v %s", contentID, bagitID, overview.Name)))
		return
	}
	tpl := s.templates["content.gohtml"]
	if err := tpl.Execute(w, struct {
		Institution, Logo string
		Content           *Content
	}{
		Institution: s.institution,
		Logo:        s.logo,
		Content:     content,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(fmt.Sprintf("error executing template %s : %v", "bagit.gohtml", err)))
		return
	}
}
