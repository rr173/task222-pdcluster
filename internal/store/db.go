// Package store 提供基于 SQLite（modernc.org/sqlite，纯 Go 驱动，CGO 无关）的
// 持久化实现：建表迁移与试验/相位参考/通道/脉冲/相位簇/解释/快照的 CRUD 及统计查询。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Open 打开（或创建）SQLite 数据库并执行迁移。
// 支持 ":memory:" 用于测试与冒烟验证。
func Open(path string) (*DB, error) {
	if path == ":memory:" {
		db, err := sql.Open("sqlite", "file::memory:?cache=shared")
		if err != nil {
			return nil, fmt.Errorf("open memory db: %w", err)
		}
		db.SetMaxOpenConns(1)
		d := &DB{db: db, path: path}
		if err := d.migrate(); err != nil {
			db.Close()
			return nil, err
		}
		return d, nil
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable fk: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	d := &DB{db: db, path: path}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// DB 封装 SQLite 连接。
type DB struct {
	db   *sql.DB
	path string
}

// Close 关闭数据库连接。
func (d *DB) Close() error { return d.db.Close() }

// Path 返回数据库路径（调试用）。
func (d *DB) Path() string { return d.path }

// SQL 暴露底层连接（供 Store 实现使用）。
func (d *DB) SQL() *sql.DB { return d.db }

// migrate 建表：全部业务表 + 唯一约束（防重复）+ 索引。
func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trials (
			id            TEXT PRIMARY KEY,
			code          TEXT NOT NULL,
			cable_name    TEXT NOT NULL,
			voltage_class TEXT NOT NULL,
			status        TEXT NOT NULL,
			fingerprint   TEXT NOT NULL,
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_trials_fingerprint ON trials(fingerprint)`,
		`CREATE TABLE IF NOT EXISTS phase_references (
			id           TEXT PRIMARY KEY,
			trial_id     TEXT NOT NULL REFERENCES trials(id),
			freq_hz      REAL NOT NULL,
			zero_time_ns INTEGER NOT NULL,
			created_at   TEXT NOT NULL,
			UNIQUE(trial_id)
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id         TEXT PRIMARY KEY,
			trial_id   TEXT NOT NULL REFERENCES trials(id),
			name       TEXT NOT NULL,
			idx        INTEGER NOT NULL,
			delay_ns   REAL NOT NULL DEFAULT 0,
			status     TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, idx)
		)`,
		`CREATE TABLE IF NOT EXISTS pulses (
			id             TEXT PRIMARY KEY,
			trial_id       TEXT NOT NULL REFERENCES trials(id),
			channel_id     TEXT NOT NULL,
			channel_index  INTEGER NOT NULL,
			seq            INTEGER NOT NULL,
			time_ns        INTEGER NOT NULL,
			amplitude_mv   REAL NOT NULL,
			phase_deg      REAL NOT NULL DEFAULT -1,
			status         TEXT NOT NULL,
			exclude_reason TEXT NOT NULL DEFAULT '',
			created_at     TEXT NOT NULL,
			UNIQUE(trial_id, channel_index, seq)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pulses_trial ON pulses(trial_id)`,
		`CREATE TABLE IF NOT EXISTS clusters (
			id               TEXT PRIMARY KEY,
			trial_id         TEXT NOT NULL REFERENCES trials(id),
			phase_start_deg  REAL NOT NULL,
			phase_end_deg    REAL NOT NULL,
			phase_center_deg REAL NOT NULL,
			pulse_count      INTEGER NOT NULL,
			max_amplitude_mv REAL NOT NULL,
			avg_amplitude_mv REAL NOT NULL,
			status           TEXT NOT NULL,
			created_at       TEXT NOT NULL,
			updated_at       TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_clusters_trial ON clusters(trial_id)`,
		`CREATE TABLE IF NOT EXISTS interpretations (
			id          TEXT PRIMARY KEY,
			trial_id    TEXT NOT NULL REFERENCES trials(id),
			defect_type TEXT NOT NULL,
			confidence  REAL NOT NULL,
			evidence    TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL,
			created_at  TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_interpretations_trial ON interpretations(trial_id)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id           TEXT PRIMARY KEY,
			trial_id     TEXT NOT NULL REFERENCES trials(id),
			version      INTEGER NOT NULL,
			status       TEXT NOT NULL,
			state_json   TEXT NOT NULL DEFAULT '{}',
			summary      TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			published_at TEXT,
			UNIQUE(trial_id, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_trial ON snapshots(trial_id)`,
	}
	for _, s := range stmts {
		if _, err := d.db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w (%s)", err, firstWords(s, 10))
		}
	}
	return nil
}

func firstWords(s string, n int) string {
	parts := strings.Fields(s)
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, " ")
}

func nowUTC() time.Time { return time.Now().UTC() }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTS(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }

type scanner interface{ Scan(dest ...any) error }
