package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// Summarizer turns a trace step sequence into a plain-English transcript.
type Summarizer interface {
	Summarize(ctx context.Context, req TranscriptRequest) (TranscriptResult, error)
}

// TranscriptRequest is the input shape to a Summarizer call.
type TranscriptRequest struct {
	TraceID    string
	TraceName  string
	TraceSteps []snapshot.Step
}

// TranscriptResult is the structured transcript returned to the API caller.
type TranscriptResult struct {
	Summary      string   `json:"summary"`
	KeyDecisions []string `json:"key_decisions"`
}

// GetTranscript returns an HTTP handler for GET /api/traces/{id}/transcript.
func GetTranscript(database *db.DB, summarizer Summarizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		trace, err := database.GetTrace(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load trace")
			return
		}

		promptsHash := transcriptPromptsHash(trace.Steps)

		if cached, err := database.GetTranscriptCache(trace.ID, promptsHash); err == nil {
			writeTranscriptJSON(w, TranscriptResult{
				Summary:      cached.Summary,
				KeyDecisions: cached.KeyDecisions,
			})
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			errorResponse(w, http.StatusInternalServerError, "failed to read transcript cache")
			return
		}

		req := TranscriptRequest{
			TraceID:    trace.ID,
			TraceName:  trace.Name,
			TraceSteps: trace.Steps,
		}
		result, err := summarizer.Summarize(r.Context(), req)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "summarize failed")
			return
		}

		if err := database.PutTranscriptCache(trace.ID, promptsHash, db.TranscriptCacheRow{
			Summary:      result.Summary,
			KeyDecisions: result.KeyDecisions,
		}); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to write transcript cache")
			return
		}

		writeTranscriptJSON(w, result)
	}
}

// writeTranscriptJSON serializes a TranscriptResult as the 200 response body.
func writeTranscriptJSON(w http.ResponseWriter, result TranscriptResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// transcriptPromptsHash builds the cache discriminator from the structural prompt inputs.
// Step content + role + tool-call shape is the structural surface; trace metadata (name,
// id, created_at) is caller-discretionary and excluded so that re-uploads of the same
// content under a different name still hit the cache. Reuses writeStepSig from triage.go
// since the canonical step signature is identical across summarization endpoints.
func transcriptPromptsHash(steps []snapshot.Step) string {
	h := sha256.New()
	h.Write([]byte(transcriptPromptVersion))
	writeStepSig(h, steps)
	return hex.EncodeToString(h.Sum(nil))
}

// transcriptPromptVersion bumps when the prompt template changes; invalidates all caches.
const transcriptPromptVersion = "v1"
