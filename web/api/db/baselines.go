package db

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Baseline represents a stored baseline row.
type Baseline struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// BaselineSummary is a lightweight baseline listing with trace count.
type BaselineSummary struct {
	ID         string
	Name       string
	TraceCount int
	CreatedAt  time.Time
}

// CreateBaseline inserts a baseline and associates the given trace IDs.
func (db *DB) CreateBaseline(name string, traceIDs []string) (Baseline, error) {
	id := uuid.New().String()

	tx, err := db.conn.Begin()
	if err != nil {
		return Baseline{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO baselines (id, name) VALUES (?, ?)`, id, name)
	if err != nil {
		return Baseline{}, fmt.Errorf("insert baseline: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO baseline_traces (baseline_id, trace_id) VALUES (?, ?)`)
	if err != nil {
		return Baseline{}, fmt.Errorf("prepare baseline_traces: %w", err)
	}
	defer stmt.Close()

	for _, tid := range traceIDs {
		if _, err := stmt.Exec(id, tid); err != nil {
			return Baseline{}, fmt.Errorf("insert baseline_trace: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Baseline{}, fmt.Errorf("commit: %w", err)
	}

	return Baseline{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

// ListBaselines returns all baselines with their trace counts.
func (db *DB) ListBaselines() ([]BaselineSummary, error) {
	rows, err := db.conn.Query(`
		SELECT b.id, b.name,
			(SELECT COUNT(*) FROM baseline_traces bt WHERE bt.baseline_id = b.id) AS trace_count,
			b.created_at
		FROM baselines b
		ORDER BY b.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list baselines: %w", err)
	}
	defer rows.Close()

	var baselines []BaselineSummary
	for rows.Next() {
		var b BaselineSummary
		if err := rows.Scan(&b.ID, &b.Name, &b.TraceCount, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan baseline: %w", err)
		}
		baselines = append(baselines, b)
	}
	return baselines, rows.Err()
}

// GetBaselineTraces returns all traces (with their steps) for a baseline.
func (db *DB) GetBaselineTraces(baselineID string) ([]TraceDetail, error) {
	rows, err := db.conn.Query(`
		SELECT t.id FROM traces t
		INNER JOIN baseline_traces bt ON bt.trace_id = t.id
		WHERE bt.baseline_id = ?
	`, baselineID)
	if err != nil {
		return nil, fmt.Errorf("get baseline traces: %w", err)
	}
	defer rows.Close()

	var traceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan trace id: %w", err)
		}
		traceIDs = append(traceIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var details []TraceDetail
	for _, id := range traceIDs {
		td, err := db.GetTrace(id)
		if err != nil {
			return nil, fmt.Errorf("get trace %s: %w", id, err)
		}
		details = append(details, td)
	}
	return details, nil
}
