package db

import (
	"encoding/json"
	"fmt"
)

// Embedding is one row in the trace_embeddings table.
type Embedding struct {
	TraceID   string
	Vector    []float32
	ModelName string
}

// InsertEmbedding writes (or replaces) the embedding row for a trace.
// trace_id is the table's primary key, so re-running the embedder on the same
// trace upserts rather than inserts duplicates.
func (db *DB) InsertEmbedding(traceID string, vector []float32, modelName string) error {
	raw, err := json.Marshal(vector)
	if err != nil {
		return fmt.Errorf("marshal embedding vector: %w", err)
	}
	_, err = db.conn.Exec(
		`INSERT INTO trace_embeddings (trace_id, vector_json, model_name)
		 VALUES (?, ?, ?)
		 ON CONFLICT(trace_id) DO UPDATE SET
		   vector_json = excluded.vector_json,
		   model_name  = excluded.model_name,
		   created_at  = CURRENT_TIMESTAMP`,
		traceID, string(raw), modelName,
	)
	if err != nil {
		return fmt.Errorf("insert trace_embedding: %w", err)
	}
	return nil
}

// GetEmbedding returns the embedding row for the given trace. Returns
// sql.ErrNoRows when no embedding has been generated for the trace.
func (db *DB) GetEmbedding(traceID string) (Embedding, error) {
	var emb Embedding
	var vectorJSON string
	err := db.conn.QueryRow(
		`SELECT trace_id, vector_json, model_name
		 FROM trace_embeddings WHERE trace_id = ?`,
		traceID,
	).Scan(&emb.TraceID, &vectorJSON, &emb.ModelName)
	if err != nil {
		return Embedding{}, err
	}
	if err := json.Unmarshal([]byte(vectorJSON), &emb.Vector); err != nil {
		return Embedding{}, fmt.Errorf("unmarshal embedding vector: %w", err)
	}
	return emb, nil
}

// ListEmbeddings returns every row in trace_embeddings. Used by the similarity
// search handler to score all candidates against the source trace.
func (db *DB) ListEmbeddings() ([]Embedding, error) {
	rows, err := db.conn.Query(
		`SELECT trace_id, vector_json, model_name FROM trace_embeddings`,
	)
	if err != nil {
		return nil, fmt.Errorf("list trace_embeddings: %w", err)
	}
	defer rows.Close()

	var out []Embedding
	for rows.Next() {
		var emb Embedding
		var vectorJSON string
		if err := rows.Scan(&emb.TraceID, &vectorJSON, &emb.ModelName); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(vectorJSON), &emb.Vector); err != nil {
			return nil, fmt.Errorf("unmarshal embedding vector for %s: %w", emb.TraceID, err)
		}
		out = append(out, emb)
	}
	return out, rows.Err()
}
