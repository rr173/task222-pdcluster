package httpapi

import (
	"net/http"
)

// createTrialRequest 是创建试验的请求体。
type createTrialRequest struct {
	Code         string `json:"code"`
	CableName    string `json:"cable_name"`
	VoltageClass string `json:"voltage_class"`
}

func (s *Server) handleCreateTrial(w http.ResponseWriter, r *http.Request) {
	var req createTrialRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	t, err := s.app.Trials.Create(req.Code, req.CableName, req.VoltageClass)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleListTrials(w http.ResponseWriter, _ *http.Request) {
	trials, err := s.app.Trials.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trials)
}

func (s *Server) handleGetTrial(w http.ResponseWriter, r *http.Request) {
	t, err := s.app.Trials.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleStartTrial(w http.ResponseWriter, r *http.Request) {
	t, err := s.app.Trials.StartAcquisition(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleFinishTrial(w http.ResponseWriter, r *http.Request) {
	t, err := s.app.Trials.FinishAcquisition(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleReviewTrial(w http.ResponseWriter, r *http.Request) {
	t, err := s.app.Trials.Review(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleSealTrial(w http.ResponseWriter, r *http.Request) {
	t, err := s.app.Trials.Seal(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// setReferenceRequest 是设置相位参考的请求体。
type setReferenceRequest struct {
	FreqHz     float64 `json:"freq_hz"`
	ZeroTimeNs int64   `json:"zero_time_ns"`
}

func (s *Server) handleSetReference(w http.ResponseWriter, r *http.Request) {
	var req setReferenceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	ref, err := s.app.Trials.SetReference(r.PathValue("id"), req.FreqHz, req.ZeroTimeNs)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ref)
}

func (s *Server) handleGetReference(w http.ResponseWriter, r *http.Request) {
	ref, err := s.app.Trials.GetReference(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ref)
}

// addChannelRequest 是登记通道的请求体。
type addChannelRequest struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
}

func (s *Server) handleAddChannel(w http.ResponseWriter, r *http.Request) {
	var req addChannelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.app.Trials.AddChannel(r.PathValue("id"), req.Name, req.Index)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.app.Trials.ListChannels(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, channels)
}
