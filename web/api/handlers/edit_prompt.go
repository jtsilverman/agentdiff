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

// EditPrompter re-runs an agent from a midpoint with the user/system prompt at
// that step rewritten, returning the continuation step sequence.
type EditPrompter interface {
	Replay(ctx context.Context, req EditPromptRequest) (EditPromptResult, error)
}

// EditPromptRequest is the input shape to an EditPrompter call.
type EditPromptRequest struct {
	OriginalTraceID   string
	OriginalTraceName string
	StepsBefore       []snapshot.Step
	ModifiedPrompt    string
	StepIndex         int
}

// EditPromptResult is the agent continuation produced by an EditPrompter.
// NewSteps is appended after the prefix + modified-prompt step to form the
// edited trace.
type EditPromptResult struct {
	NewSteps []snapshot.Step
}

// editPromptRequest is the POST body shape for /api/traces/:id/edit-prompt.
type editPromptRequest struct {
	StepIndex      int    `json:"step_index"`
	ModifiedPrompt string `json:"modified_prompt"`
}

// editPromptResponse is the success response body. Shape is identical to
// counterfactualResponse so the frontend can reuse CounterfactualGraph.
type editPromptResponse struct {
	NewTraceID string                   `json:"new_trace_id"`
	Comparison counterfactualComparison `json:"comparison"`
}

// PostEditPrompt handles POST /api/traces/:id/edit-prompt.
// Body: {"step_index": N, "modified_prompt": "..."}.
// Replays the agent from step N with the prompt at that step substituted; persists
// the result as a new trace + a prompt_edit_runs row linking it to the original.
// Even when the EditPrompter fails, returns 201 with a synthetic minimal trace
// that records the failure mode — never bubbles LLM errors to the caller.
func PostEditPrompt(database *db.DB, ep EditPrompter) http.HandlerFunc {
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
		var req editPromptRequest
		if err := json.Unmarshal(body, &req); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if req.StepIndex < 0 || req.StepIndex >= len(original.Steps) {
			errorResponse(w, http.StatusBadRequest, "step_index out of range")
			return
		}

		stepsBefore := make([]snapshot.Step, req.StepIndex)
		copy(stepsBefore, original.Steps[:req.StepIndex])

		epReq := EditPromptRequest{
			OriginalTraceID:   original.ID,
			OriginalTraceName: original.Name,
			StepsBefore:       stepsBefore,
			ModifiedPrompt:    req.ModifiedPrompt,
			StepIndex:         req.StepIndex,
		}

		var continuation []snapshot.Step
		result, epErr := ep.Replay(r.Context(), epReq)
		if epErr != nil {
			continuation = []snapshot.Step{
				{
					Role:    "assistant",
					Content: "Prompt-edit replay unavailable: " + epErr.Error(),
				},
			}
		} else {
			continuation = result.NewSteps
		}

		// Assemble the edited trace's full step sequence:
		// prefix (steps before step_index) + the modified-prompt step + LLM continuation.
		// The modified-prompt step uses role="user" since a prompt edit is the user
		// rewriting what they asked the agent at that midpoint.
		modifiedStep := snapshot.Step{
			Role:    "user",
			Content: req.ModifiedPrompt,
		}
		newSteps := make([]snapshot.Step, 0, len(stepsBefore)+1+len(continuation))
		newSteps = append(newSteps, stepsBefore...)
		newSteps = append(newSteps, modifiedStep)
		newSteps = append(newSteps, continuation...)

		newName := "prompt-edit-" + original.Name
		newTrace, err := database.CreateTrace(newName, original.Adapter, nil)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to create edited trace")
			return
		}
		if err := database.InsertSnapshots(newTrace.ID, newSteps); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to persist edited steps")
			return
		}
		if _, err := database.CreatePromptEditRun(original.ID, newTrace.ID, req.StepIndex, req.ModifiedPrompt); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to link prompt-edit run")
			return
		}

		comparison := counterfactualComparison{
			OriginalPath:   stepPath(original.Steps),
			NewPath:        stepPath(newSteps),
			DivergenceStep: req.StepIndex,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(editPromptResponse{
			NewTraceID: newTrace.ID,
			Comparison: comparison,
		})
	}
}
