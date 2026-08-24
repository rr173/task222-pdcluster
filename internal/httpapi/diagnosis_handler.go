package httpapi

import (
	"net/http"
)

func (s *Server) handleMarkInterference(w http.ResponseWriter, r *http.Request) {
	c, err := s.app.Diagnosis.MarkInterference(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleConfirmCluster(w http.ResponseWriter, r *http.Request) {
	c, err := s.app.Diagnosis.ConfirmCluster(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleListInterpretations(w http.ResponseWriter, r *http.Request) {
	interps, err := s.app.Diagnosis.ListInterpretations(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, interps)
}
