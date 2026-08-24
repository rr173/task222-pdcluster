package store

import "database/sql"

// Store 聚合所有子 Store，供 service 层注入。
type Store struct {
	Trials          *TrialStore
	References      *ReferenceStore
	Channels        *ChannelStore
	Pulses          *PulseStore
	Clusters        *ClusterStore
	Interpretations *InterpretationStore
	Snapshots       *SnapshotStore
	Stats           *StatsStore
}

// NewStore 用底层 *sql.DB 构造聚合 Store。
func NewStore(db *sql.DB) *Store {
	return &Store{
		Trials:          &TrialStore{db: db},
		References:      &ReferenceStore{db: db},
		Channels:        &ChannelStore{db: db},
		Pulses:          &PulseStore{db: db},
		Clusters:        &ClusterStore{db: db},
		Interpretations: &InterpretationStore{db: db},
		Snapshots:       &SnapshotStore{db: db},
		Stats:           &StatsStore{db: db},
	}
}
