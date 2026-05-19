package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// Counterfactualer re-runs an agent from a midpoint with a modified input,
// returning the continuation step sequence.
type Counterfactualer interface {
	Replay(ctx context.Context, req CounterfactualRequest) (CounterfactualResult, error)
}

// CounterfactualRequest is the input shape to a Counterfactualer call.
type CounterfactualRequest struct {
	OriginalTraceID   string
	OriginalTraceName string
	StepsBefore       []snapshot.Step
	ModifiedInput     string
	StepIndex         int
}

// CounterfactualResult is the agent continuation produced by a Counterfactualer.
// NewSteps is appended after the prefix + modified-input step to form the
// counterfactual trace.
type CounterfactualResult struct {
	NewSteps []snapshot.Step
}

// counterfactualRequest is the POST body shape for /api/traces/:id/counterfactual.
type counterfactualRequest struct {
	StepIndex     int    `json:"step_index"`
	ModifiedInput string `json:"modified_input"`
}

// counterfactualResponse is the success response body.
type counterfactualResponse struct {
	NewTraceID string                  `json:"new_trace_id"`
	Comparison counterfactualComparison `json:"comparison"`
}

// counterfactualComparison summarizes the two paths and where they diverged.
// original_path and new_path are tool-name (or role if no tool) sequences;
// divergence_step is the index at which they first differ.
type counterfactualComparison struct {
	OriginalPath   []string `json:"original_path"`
	NewPath        []string `json:"new_path"`
	DivergenceStep int      `json:"divergence_step"`
}

// PostCounterfactual handles POST /api/traces/:id/counterfactual.
// Body: {"step_index": N, "modified_input": "..."}.
// Replays the agent from step N with the modified input substituted; persists
// the result as a new trace + a counterfactual_runs row linking it to the original.
// Even when the Counterfactualer fails, returns 201 with a synthetic minimal trace
// that records the failure mode — never bubbles LLM errors to the caller.
func PostCounterfactual(database *db.DB, cf Counterfactualer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		originalID := chi.URLParam(r, "id")

		original, err := database.GetTrace(originalID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load trace")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		defer r.Body.Close()
		var req counterfactualRequest
		if err := json.Unmarshal(body, &req); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if req.StepIndex < 0 || req.StepIndex >= len(original.Steps) {
			errorResponse(w, http.StatusBadRequest, "step_index out of range")
			return
		}

		// Build the request to the Counterfactualer. StepsBefore is the prefix the
		// agent had seen before the step we're replacing — that's the dynamic context
		// the LLM needs to continue plausibly from this midpoint.
		stepsBefore := make([]snapshot.Step, req.StepIndex)
		copy(stepsBefore, original.Steps[:req.StepIndex])

		cfReq := CounterfactualRequest{
			OriginalTraceID:   original.ID,
			OriginalTraceName: original.Name,
			StepsBefore:       stepsBefore,
			ModifiedInput:     req.ModifiedInput,
			StepIndex:         req.StepIndex,
		}

		var continuation []snapshot.Step
		result, cfErr := cf.Replay(r.Context(), cfReq)
		if cfErr != nil {
			// Deterministic fallback: synthetic continuation recording the failure.
			// The contract holds — caller gets a parseable new trace even when the LLM
			// call is unavailable.
			continuation = []snapshot.Step{
				{
					Role:    "assistant",
					Content: "Counterfactual replay unavailable: " + cfErr.Error(),
				},
			}
		} else {
			continuation = result.NewSteps
		}

		// Assemble the counterfactual trace's full step sequence:
		// prefix (steps before step_index) + the modified-input step + LLM continuation.
		modifiedStep := snapshot.Step{
			Role:    "user",
			Content: req.ModifiedInput,
		}
		newSteps := make([]snapshot.Step, 0, len(stepsBefore)+1+len(continuation))
		newSteps = append(newSteps, stepsBefore...)
		newSteps = append(newSteps, modifiedStep)
		newSteps = append(newSteps, continuation...)

		newName := "counterfactual-" + original.Name
		newTrace, err := database.CreateTrace(newName, original.Adapter, nil)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to create counterfactual trace")
			return
		}
		if err := database.InsertSnapshots(newTrace.ID, newSteps); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to persist counterfactual steps")
			return
		}
		if _, err := database.CreateCounterfactualRun(original.ID, newTrace.ID, req.StepIndex, req.ModifiedInput); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to link counterfactual run")
			return
		}

		comparison := counterfactualComparison{
			OriginalPath:   stepPath(original.Steps),
			NewPath:        stepPath(newSteps),
			DivergenceStep: req.StepIndex,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(counterfactualResponse{
			NewTraceID: newTrace.ID,
			Comparison: comparison,
		})
	}
}

// stepPath reduces a step sequence to a list of identifiers used by the comparison
// object: tool name if the step is a tool call/result, otherwise the role.
// Same shape on both sides so the UI can render a divergence at the same indices.
func stepPath(steps []snapshot.Step) []string {
	path := make([]string, len(steps))
	for i, s := range steps {
		switch {
		case s.ToolCall != nil:
			path[i] = s.ToolCall.Name
		case s.ToolResult != nil:
			path[i] = s.ToolResult.Name
		default:
			path[i] = s.Role
		}
	}
	return path
}
