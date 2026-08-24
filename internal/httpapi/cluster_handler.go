package httpapi

import (
	"net/http"
)

func (s *Server) handleCluster(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.app.Clusters.Cluster(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, clusters)
}

func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.app.Clusters.ListClusters(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, clusters)
}

func (s *Server) handleGetCluster(w http.ResponseWriter, r *http.Request) {
	c, err := s.app.Clusters.GetCluster(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// mergeClustersRequest 是合并两个簇的请求体。
type mergeClustersRequest struct {
	ClusterA string `json:"cluster_a"`
	ClusterB string `json:"cluster_b"`
}

func (s *Server) handleMergeClusters(w http.ResponseWriter, r *http.Request) {
	var req mergeClustersRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	merged, err := s.app.Clusters.MergeClusters(req.ClusterA, req.ClusterB)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, merged)
}
