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
    tool_is_error INTEGER NOT NULL DEFAULT 0,
    cost_tokens INTEGER,
    latency_ms  INTEGER
);

CREATE TABLE IF NOT EXISTS baselines (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS baseline_traces (
    baseline_id TEXT NOT NULL REFERENCES baselines(id),
    trace_id    TEXT NOT NULL REFERENCES traces(id),
    PRIMARY KEY (baseline_id, trace_id)
);

CREATE TABLE IF NOT EXISTS triage_cache (
    trace_a_id     TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    trace_b_id     TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    prompts_hash   TEXT NOT NULL,
    summary        TEXT NOT NULL,
    classification TEXT NOT NULL,
    likely_cause   TEXT NOT NULL,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (trace_a_id, trace_b_id, prompts_hash)
);

CREATE TABLE IF NOT EXISTS transcripts (
    trace_id      TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    prompts_hash  TEXT NOT NULL,
    summary       TEXT NOT NULL,
    key_decisions TEXT NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (trace_id, prompts_hash)
);

CREATE TABLE IF NOT EXISTS counterfactual_runs (
    id                      TEXT PRIMARY KEY,
    original_trace_id       TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    counterfactual_trace_id TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    step_index              INTEGER NOT NULL,
    modified_input          TEXT NOT NULL,
    created_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompt_edit_runs (
    id                TEXT PRIMARY KEY,
    original_trace_id TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    edited_trace_id   TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    step_index        INTEGER NOT NULL,
    modified_prompt   TEXT NOT NULL,
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS trace_embeddings (
    trace_id     TEXT PRIMARY KEY REFERENCES traces(id) ON DELETE CASCADE,
    vector_json  TEXT NOT NULL,
    model_name   TEXT NOT NULL,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS demo_rate_limits (
    ip    TEXT NOT NULL,
    day   TEXT NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (ip, day)
);

CREATE INDEX IF NOT EXISTS idx_snapshots_trace ON snapshots(trace_id, step_index);
CREATE INDEX IF NOT EXISTS idx_baseline_traces_baseline ON baseline_traces(baseline_id);
CREATE INDEX IF NOT EXISTS idx_counterfactual_runs_original ON counterfactual_runs(original_trace_id);
CREATE INDEX IF NOT EXISTS idx_prompt_edit_runs_original ON prompt_edit_runs(original_trace_id);
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
	// Additive nullable-column migrations for DBs created before the column
	// existed in the CREATE TABLE schema. PRAGMA table_info is used over
	// ALTER TABLE ADD COLUMN IF NOT EXISTS so we don't depend on SQLite 3.35+,
	// and over try/catch on constraint errors which would mask real failures.
	if err := migrateAddColumn(conn, "snapshots", "cost_tokens", "INTEGER"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := migrateAddColumn(conn, "snapshots", "latency_ms", "INTEGER"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := migrateAddColumn(conn, "baselines", "description", "TEXT NOT NULL DEFAULT ''"); err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{conn: conn}, nil
}

// migrateAddColumn adds a nullable column to an existing table if PRAGMA
// table_info shows it is missing. No-op when the column already exists, so
// it is safe to call on every NewDB.
func migrateAddColumn(conn *sql.DB, table, column, sqlType string) error {
	rows, err := conn.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = conn.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + sqlType)
	return err
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
