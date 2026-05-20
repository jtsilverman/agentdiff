package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/internal/embed"
)

// similarMatchResponse is one row in the GET /api/traces/:id/similar envelope.
type similarMatchResponse struct {
	TraceID         string  `json:"trace_id"`
	Name            string  `json:"name"`
	SimilarityScore float64 `json:"similarity_score"`
}

// similarResponse is the envelope returned by GET /api/traces/:id/similar.
type similarResponse struct {
	Matches []similarMatchResponse `json:"matches"`
}

// topKSimilar caps how many matches the endpoint returns.
const topKSimilar = 5

// GetSimilar handles GET /api/traces/:id/similar.
// Reads the source trace's embedding, scores every other embedding by cosine,
// returns the top-5 sorted by similarity descending. Source is excluded from
// its own result set; unembedded traces are skipped naturally (no row in
// trace_embeddings means no comparison candidate).
func GetSimilar(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := chi.URLParam(r, "id")

		// 404 if the trace itself doesn't exist (separate from "trace exists but
		// has no embedding yet"). GetTrace is the canonical existence check.
		if _, err := database.GetTrace(traceID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load trace")
			return
		}

		// Source has no embedding → return empty matches gracefully.
		// The /similar endpoint is the read side; absence of an embedding row
		// is expected when the embedder is nil or the Voyage call failed.
		sourceEmb, err := database.GetEmbedding(traceID)
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusOK, similarResponse{Matches: []similarMatchResponse{}})
			return
		}
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to load source embedding")
			return
		}

		// Score every other embedding by cosine against the source.
		all, err := database.ListEmbeddings()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to list embeddings")
			return
		}

		type scored struct {
			traceID string
			score   float64
		}
		var pool []scored
		for _, e := range all {
			if e.TraceID == sourceEmb.TraceID {
				continue
			}
			s, err := embed.Cosine(sourceEmb.Vector, e.Vector)
			if err != nil {
				// Skip a single bad row rather than fail the whole request —
				// dim mismatches between models are recoverable per-row.
				continue
			}
			pool = append(pool, scored{traceID: e.TraceID, score: s})
		}

		sort.Slice(pool, func(i, j int) bool {
			return pool[i].score > pool[j].score
		})

		if len(pool) > topKSimilar {
			pool = pool[:topKSimilar]
		}

		// Resolve trace names for the surviving top-K. Skipping ListTraces here
		// because the candidate set is small (≤ 5); a per-row GetTrace is fine.
		matches := make([]similarMatchResponse, 0, len(pool))
		for _, p := range pool {
			tr, err := database.GetTrace(p.traceID)
			if err != nil {
				continue
			}
			matches = append(matches, similarMatchResponse{
				TraceID:         tr.ID,
				Name:            tr.Name,
				SimilarityScore: p.score,
			})
		}

		writeJSON(w, http.StatusOK, similarResponse{Matches: matches})
	}
}

// writeJSON is a tiny helper used by the chunk-22 handlers. Other handlers in
// this package open-code the same pattern; this exists so similar.go isn't
// noisy with header + WriteHeader + Encode triples.
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
