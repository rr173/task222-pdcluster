package store

import (
	"database/sql"
	"fmt"
)

// StatsStore 提供跨表统计（用于 /api/stats 与自检）。
type StatsStore struct{ db *sql.DB }

// Stats 汇总各实体的数量。
type Stats struct {
	Trials         int `json:"trials"`
	Channels       int `json:"channels"`
	Pulses         int `json:"pulses"`
	ValidPulses    int `json:"valid_pulses"`
	Clusters       int `json:"clusters"`
	Interpretations int `json:"interpretations"`
	Snapshots      int `json:"snapshots"`
}

// Count 返回全库统计。
func (s *StatsStore) Count() (*Stats, error) {
	st := &Stats{}
	queries := []struct {
		field *int
		sql   string
	}{
		{&st.Trials, `SELECT COUNT(*) FROM trials`},
		{&st.Channels, `SELECT COUNT(*) FROM channels`},
		{&st.Pulses, `SELECT COUNT(*) FROM pulses`},
		{&st.ValidPulses, `SELECT COUNT(*) FROM pulses WHERE status = 'valid'`},
		{&st.Clusters, `SELECT COUNT(*) FROM clusters`},
		{&st.Interpretations, `SELECT COUNT(*) FROM interpretations`},
		{&st.Snapshots, `SELECT COUNT(*) FROM snapshots`},
	}
	for _, q := range queries {
		if err := s.db.QueryRow(q.sql).Scan(q.field); err != nil {
			return nil, fmt.Errorf("stats: %w", err)
		}
	}
	return st, nil
}
