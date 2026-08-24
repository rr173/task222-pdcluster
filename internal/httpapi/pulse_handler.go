package httpapi

import (
	"net/http"

	"task222-pdcluster/internal/service"
)

// ingestPulseRequest 是接收单个脉冲的请求体。
type ingestPulseRequest struct {
	ChannelID    string  `json:"channel_id"`
	ChannelIndex int     `json:"channel_index"`
	Seq          int64   `json:"seq"`
	TimeNs       int64   `json:"time_ns"`
	AmplitudeMv  float64 `json:"amplitude_mv"`
}

func (s *Server) handleIngestPulse(w http.ResponseWriter, r *http.Request) {
	var req ingestPulseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	res, err := s.app.Pulses.Ingest(r.PathValue("id"), req.ChannelID, req.ChannelIndex, req.Seq, req.TimeNs, req.AmplitudeMv)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// ingestBatchRequest 是批量接收脉冲的请求体。
type ingestBatchRequest struct {
	Pulses []ingestPulseRequest `json:"pulses"`
}

func (s *Server) handleIngestBatch(w http.ResponseWriter, r *http.Request) {
	var req ingestBatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	trialID := r.PathValue("id")
	inputs := make([]service.PulseInput, 0, len(req.Pulses))
	for _, p := range req.Pulses {
		inputs = append(inputs, service.PulseInput{ChannelID: p.ChannelID, ChannelIndex: p.ChannelIndex, Seq: p.Seq, TimeNs: p.TimeNs, AmplitudeMv: p.AmplitudeMv})
	}
	result, err := s.app.Pulses.IngestBatch(trialID, inputs)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleListPulses(w http.ResponseWriter, r *http.Request) {
	pulses, err := s.app.Pulses.ListPulses(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pulses)
}

func (s *Server) handleCalibrate(w http.ResponseWriter, r *http.Request) {
	delays, err := s.app.Pulses.Calibrate(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel_delays_ns": delays})
}

func (s *Server) handleGetCalibration(w http.ResponseWriter, r *http.Request) {
	channels, err := s.app.Trials.ListChannels(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, channels)
}
