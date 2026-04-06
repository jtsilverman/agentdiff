package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// diffTraceRef identifies a trace in the diff response.
type diffTraceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// diffStepJSON is the JSON representation of a step in a diff pair.
type diffStepJSON struct {
	Role       string              `json:"role"`
	Content    string              `json:"content"`
	ToolCall   *toolCallResponse   `json:"tool_call,omitempty"`
	ToolResult *toolResultResponse `json:"tool_result,omitempty"`
}

// diffPair represents one aligned pair in the diff output.
type diffPair struct {
	AStep *diffStepJSON `json:"a_step"`
	BStep *diffStepJSON `json:"b_step"`
	Op    string        `json:"op"`
}

// diffSummary counts each operation type in the alignment.
type diffSummary struct {
	Matches       int `json:"matches"`
	Insertions    int `json:"insertions"`
	Deletions     int `json:"deletions"`
	Substitutions int `json:"substitutions"`
}

// diffResponse is the JSON response for GET /api/diff/{idA}/{idB}.
type diffResponse struct {
	TraceA    diffTraceRef `json:"trace_a"`
	TraceB    diffTraceRef `json:"trace_b"`
	Alignment []diffPair   `json:"alignment"`
	Distance  int          `json:"distance"`
	Summary   diffSummary  `json:"summary"`
}

// opLabel converts an AlignOp enum to its string label.
func opLabel(op diff.AlignOp) string {
	switch op {
	case diff.AlignMatch:
		return "match"
	case diff.AlignSubst:
		return "substitute"
	case diff.AlignInsert:
		return "insert"
	case diff.AlignDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// toDiffStep converts a snapshot.Step to a diffStepJSON.
func toDiffStep(s snapshot.Step) *diffStepJSON {
	ds := &diffStepJSON{
		Role:    s.Role,
		Content: s.Content,
	}
	if s.ToolCall != nil {
		ds.ToolCall = &toolCallResponse{
			Name: s.ToolCall.Name,
			Args: s.ToolCall.Args,
		}
	}
	if s.ToolResult != nil {
		ds.ToolResult = &toolResultResponse{
			Name:    s.ToolResult.Name,
			Output:  s.ToolResult.Output,
			IsError: s.ToolResult.IsError,
		}
	}
	return ds
}

// extractToolCallSteps returns only steps that have a ToolCall, preserving order.
func extractToolCallSteps(steps []snapshot.Step) []snapshot.Step {
	var out []snapshot.Step
	for _, s := range steps {
		if s.ToolCall != nil {
			out = append(out, s)
		}
	}
	return out
}

// GetDiff handles GET /api/diff/{idA}/{idB}.
func GetDiff(database *db.DB) http.HandlerFunc {
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

		// Extract tool-call-only subsequences for alignment indexing.
		toolStepsA := extractToolCallSteps(traceA.Steps)
		toolStepsB := extractToolCallSteps(traceB.Steps)

		seqA := diff.ExtractToolNames(traceA.Steps)
		seqB := diff.ExtractToolNames(traceB.Steps)

		alignment := diff.Align(seqA, seqB)
		distance := diff.Levenshtein(seqA, seqB)

		// Build diff pairs. IndexA/IndexB index into the tool-call-only subsequences.
		pairs := make([]diffPair, len(alignment.Pairs))
		var summary diffSummary

		for i, ap := range alignment.Pairs {
			dp := diffPair{
				Op: opLabel(ap.Op),
			}

			if ap.IndexA >= 0 && ap.IndexA < len(toolStepsA) {
				dp.AStep = toDiffStep(toolStepsA[ap.IndexA])
			}
			if ap.IndexB >= 0 && ap.IndexB < len(toolStepsB) {
				dp.BStep = toDiffStep(toolStepsB[ap.IndexB])
			}

			switch ap.Op {
			case diff.AlignMatch:
				summary.Matches++
			case diff.AlignInsert:
				summary.Insertions++
			case diff.AlignDelete:
				summary.Deletions++
			case diff.AlignSubst:
				summary.Substitutions++
			}

			pairs[i] = dp
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(diffResponse{
			TraceA:    diffTraceRef{ID: traceA.ID, Name: traceA.Name},
			TraceB:    diffTraceRef{ID: traceB.ID, Name: traceB.Name},
			Alignment: pairs,
			Distance:  distance,
			Summary:   summary,
		})
	}
}
