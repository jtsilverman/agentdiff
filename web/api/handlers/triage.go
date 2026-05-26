package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// Triager performs AI-assisted regression triage between two trace step sequences.
type Triager interface {
	Triage(ctx context.Context, req TriageRequest) (TriageResult, error)
}

// TriageRequest is the input shape to a Triager call.
type TriageRequest struct {
	TraceAID    string
	TraceBID    string
	TraceAName  string
	TraceBName  string
	TraceASteps []snapshot.Step
	TraceBSteps []snapshot.Step
}

// TriageResult is the structured triage answer returned to the API caller.
type TriageResult struct {
	Summary        string `json:"summary"`
	Classification string `json:"classification"`
	LikelyCause    string `json:"likely_cause"`
}

// GetTriage returns an HTTP handler for GET /api/diff/{idA}/{idB}/triage.
func GetTriage(database *db.DB, triager Triager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idA := chi.URLParam(r, "idA")
		idB := chi.URLParam(r, "idB")

		traceA, err := database.GetTrace(idA)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace A not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load trace A")
			return
		}

		traceB, err := database.GetTrace(idB)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace B not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load trace B")
			return
		}

		promptsHash := triagePromptsHash(traceA.Steps, traceB.Steps)

		if cached, err := database.GetTriageCache(traceA.ID, traceB.ID, promptsHash); err == nil {
			writeTriageJSON(w, TriageResult{
				Summary:        cached.Summary,
				Classification: cached.Classification,
				LikelyCause:    cached.LikelyCause,
			})
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			errorResponse(w, http.StatusInternalServerError, "failed to read triage cache")
			return
		}

		req := TriageRequest{
			TraceAID:    traceA.ID,
			TraceBID:    traceB.ID,
			TraceAName:  traceA.Name,
			TraceBName:  traceB.Name,
			TraceASteps: traceA.Steps,
			TraceBSteps: traceB.Steps,
		}

		result, err := triager.Triage(r.Context(), req)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "triage failed")
			return
		}

		if err := database.PutTriageCache(traceA.ID, traceB.ID, promptsHash, db.TriageCacheRow{
			Summary:        result.Summary,
			Classification: result.Classification,
			LikelyCause:    result.LikelyCause,
		}); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to write triage cache")
			return
		}

		writeTriageJSON(w, result)
	}
}

// writeTriageJSON serializes a TriageResult as the 200 response body.
func writeTriageJSON(w http.ResponseWriter, result TriageResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// TriagePromptsHash is the exported wrapper used by the seed-cache generator
// to pre-compute the same cache keys the handler computes at request time.
// Returns the discriminator used by GetTriageCache / PutTriageCache.
func TriagePromptsHash(stepsA, stepsB []snapshot.Step) string {
	return triagePromptsHash(stepsA, stepsB)
}

// triagePromptsHash builds the cache discriminator from the structural prompt inputs.
// Step content + role + tool-call shape is the structural surface; trace metadata (name,
// id, created_at) is caller-discretionary and excluded so that re-uploads of the same
// content under a different name still hit the cache.
func triagePromptsHash(stepsA, stepsB []snapshot.Step) string {
	h := sha256.New()
	h.Write([]byte(triagePromptVersion))
	writeStepSig(h, stepsA)
	h.Write([]byte("|"))
	writeStepSig(h, stepsB)
	return hex.EncodeToString(h.Sum(nil))
}

// writeStepSig writes a canonical signature of a step sequence into the hasher.
// json.Marshal sorts map keys, so ToolCall.Args (map[string]interface{}) hashes
// deterministically across runs.
func writeStepSig(h io.Writer, steps []snapshot.Step) {
	for _, s := range steps {
		fmt.Fprintf(h, "%s\x1f%s\x1f", s.Role, s.Content)
		if s.ToolCall != nil {
			fmt.Fprintf(h, "%s\x1f", s.ToolCall.Name)
			if argsJSON, err := json.Marshal(s.ToolCall.Args); err == nil {
				h.Write(argsJSON)
			}
			h.Write([]byte("\x1f"))
		}
		if s.ToolResult != nil {
			fmt.Fprintf(h, "%t\x1f%s\x1f", s.ToolResult.IsError, s.ToolResult.Output)
		}
		h.Write([]byte("\x1e"))
	}
}

// triagePromptVersion bumps when the prompt template changes; invalidates all caches.
const triagePromptVersion = "v1"
