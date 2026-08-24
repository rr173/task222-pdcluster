package httpapi

import (
	"net/http"
)

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	sn, err := s.app.Snapshots.Create(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sn)
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	snaps, err := s.app.Snapshots.ListSnapshots(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	sn, err := s.app.Snapshots.GetSnapshot(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sn)
}

func (s *Server) handlePublishSnapshot(w http.ResponseWriter, r *http.Request) {
	sn, err := s.app.Snapshots.Publish(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sn)
}
