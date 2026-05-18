package db

import (
	"encoding/json"
	"fmt"
)

// TranscriptCacheRow is one cached AI-generated transcript summary keyed by (traceID, promptsHash).
type TranscriptCacheRow struct {
	Summary      string
	KeyDecisions []string
}

// GetTranscriptCache returns the cached transcript row for the given key, or sql.ErrNoRows on miss.
func (db *DB) GetTranscriptCache(traceID, promptsHash string) (TranscriptCacheRow, error) {
	var row TranscriptCacheRow
	var decisionsJSON string
	err := db.conn.QueryRow(
		`SELECT summary, key_decisions
		 FROM transcripts
		 WHERE trace_id = ? AND prompts_hash = ?`,
		traceID, promptsHash,
	).Scan(&row.Summary, &decisionsJSON)
	if err != nil {
		return TranscriptCacheRow{}, fmt.Errorf("get transcript cache: %w", err)
	}
	if err := json.Unmarshal([]byte(decisionsJSON), &row.KeyDecisions); err != nil {
		return TranscriptCacheRow{}, fmt.Errorf("decode key_decisions: %w", err)
	}
	return row, nil
}

// PutTranscriptCache writes (or replaces) a transcript cache row for the given key.
func (db *DB) PutTranscriptCache(traceID, promptsHash string, row TranscriptCacheRow) error {
	decisionsJSON, err := json.Marshal(row.KeyDecisions)
	if err != nil {
		return fmt.Errorf("encode key_decisions: %w", err)
	}
	_, err = db.conn.Exec(
		`INSERT OR REPLACE INTO transcripts
		 (trace_id, prompts_hash, summary, key_decisions)
		 VALUES (?, ?, ?, ?)`,
		traceID, promptsHash, row.Summary, string(decisionsJSON),
	)
	if err != nil {
		return fmt.Errorf("put transcript cache: %w", err)
	}
	return nil
}
