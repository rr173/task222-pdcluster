// Package httpapi 是 HTTP 层：以 /api 为前缀暴露 JSON API，
// 每个入口把请求转交给 service 层并映射错误到状态码。
package httpapi

import (
	"net/http"

	"task222-pdcluster/internal/service"
)

// Server 封装路由与 app 依赖。
type Server struct {
	app *service.App
	mux *http.ServeMux
}

// New 构造 HTTP 服务器并注册全部路由。
func New(app *service.App) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回可交给 http.Server 的处理器。
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)

	s.mux.HandleFunc("POST /api/trials", s.handleCreateTrial)
	s.mux.HandleFunc("GET /api/trials", s.handleListTrials)
	s.mux.HandleFunc("GET /api/trials/{id}", s.handleGetTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/start", s.handleStartTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/finish", s.handleFinishTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/review", s.handleReviewTrial)
	s.mux.HandleFunc("POST /api/trials/{id}/seal", s.handleSealTrial)

	s.mux.HandleFunc("POST /api/trials/{id}/reference", s.handleSetReference)
	s.mux.HandleFunc("GET /api/trials/{id}/reference", s.handleGetReference)

	s.mux.HandleFunc("POST /api/trials/{id}/channels", s.handleAddChannel)
	s.mux.HandleFunc("GET /api/trials/{id}/channels", s.handleListChannels)

	s.mux.HandleFunc("POST /api/trials/{id}/pulses", s.handleIngestPulse)
	s.mux.HandleFunc("POST /api/trials/{id}/pulses/batch", s.handleIngestBatch)
	s.mux.HandleFunc("GET /api/trials/{id}/pulses", s.handleListPulses)

	s.mux.HandleFunc("POST /api/trials/{id}/calibrate", s.handleCalibrate)
	s.mux.HandleFunc("GET /api/trials/{id}/calibration", s.handleGetCalibration)

	s.mux.HandleFunc("POST /api/trials/{id}/cluster", s.handleCluster)
	s.mux.HandleFunc("GET /api/trials/{id}/clusters", s.handleListClusters)
	s.mux.HandleFunc("GET /api/clusters/{id}", s.handleGetCluster)
	s.mux.HandleFunc("POST /api/clusters/merge", s.handleMergeClusters)
	s.mux.HandleFunc("POST /api/clusters/{id}/interference", s.handleMarkInterference)
	s.mux.HandleFunc("POST /api/clusters/{id}/confirm", s.handleConfirmCluster)

	s.mux.HandleFunc("GET /api/trials/{id}/interpretations", s.handleListInterpretations)

	s.mux.HandleFunc("POST /api/trials/{id}/snapshots", s.handleCreateSnapshot)
	s.mux.HandleFunc("GET /api/trials/{id}/snapshots", s.handleListSnapshots)
	s.mux.HandleFunc("GET /api/snapshots/{id}", s.handleGetSnapshot)
	s.mux.HandleFunc("POST /api/snapshots/{id}/publish", s.handlePublishSnapshot)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "task222-pdcluster"})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := s.app.Stats()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
