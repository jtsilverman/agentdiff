package db

import (
	"fmt"
)

// TriageCacheRow is one cached AI-triage result keyed by (traceAID, traceBID, promptsHash).
type TriageCacheRow struct {
	Summary        string
	Classification string
	LikelyCause    string
}

// GetTriageCache returns the cached triage row for the given key, or sql.ErrNoRows on miss.
func (db *DB) GetTriageCache(traceAID, traceBID, promptsHash string) (TriageCacheRow, error) {
	var row TriageCacheRow
	err := db.conn.QueryRow(
		`SELECT summary, classification, likely_cause
		 FROM triage_cache
		 WHERE trace_a_id = ? AND trace_b_id = ? AND prompts_hash = ?`,
		traceAID, traceBID, promptsHash,
	).Scan(&row.Summary, &row.Classification, &row.LikelyCause)
	if err != nil {
		return TriageCacheRow{}, fmt.Errorf("get triage cache: %w", err)
	}
	return row, nil
}

// PutTriageCache writes (or replaces) a triage cache row for the given key.
func (db *DB) PutTriageCache(traceAID, traceBID, promptsHash string, row TriageCacheRow) error {
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO triage_cache
		 (trace_a_id, trace_b_id, prompts_hash, summary, classification, likely_cause)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		traceAID, traceBID, promptsHash, row.Summary, row.Classification, row.LikelyCause,
	)
	if err != nil {
		return fmt.Errorf("put triage cache: %w", err)
	}
	return nil
}
