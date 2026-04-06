package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a SQLite database connection for AgentDiff web storage.
type DB struct {
	conn *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS traces (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    adapter     TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT '',
    metadata    TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS snapshots (
    id          TEXT PRIMARY KEY,
    trace_id    TEXT NOT NULL REFERENCES traces(id),
    step_index  INTEGER NOT NULL,
    role        TEXT NOT NULL,
    content     TEXT,
    tool_name   TEXT,
    tool_args   TEXT,
    tool_output TEXT,
    tool_is_error INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS baselines (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS baseline_traces (
    baseline_id TEXT NOT NULL REFERENCES baselines(id),
    trace_id    TEXT NOT NULL REFERENCES traces(id),
    PRIMARY KEY (baseline_id, trace_id)
);

CREATE INDEX IF NOT EXISTS idx_snapshots_trace ON snapshots(trace_id, step_index);
CREATE INDEX IF NOT EXISTS idx_baseline_traces_baseline ON baseline_traces(baseline_id);
`

// NewDB opens a SQLite database at path and runs schema migration.
func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
