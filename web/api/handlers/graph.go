package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/internal/graph"
)

// stepsByTrace pulls each baseline trace's step list, suitable for graph.Aggregate.
func stepsByTrace(traces []db.TraceDetail) [][]snapshot.Step {
	out := make([][]snapshot.Step, len(traces))
	for i, t := range traces {
		out[i] = t.Steps
	}
	return out
}

// GetGraph handles GET /api/baselines/:id/graph.
// Returns the aggregate path graph across all traces in the baseline.
func GetGraph(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		traces, err := database.GetBaselineTraces(id)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to load baseline traces")
			return
		}
		if len(traces) == 0 {
			errorResponse(w, http.StatusNotFound, "baseline not found or has no traces")
			return
		}

		result := graph.Aggregate(stepsByTrace(traces))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// GetOverlay handles GET /api/baselines/:id/overlay/:traceID.
// Returns the overlay of the named trace against the baseline's aggregate path graph.
func GetOverlay(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		baselineID := chi.URLParam(r, "id")
		traceID := chi.URLParam(r, "traceID")

		traces, err := database.GetBaselineTraces(baselineID)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to load baseline traces")
			return
		}
		if len(traces) == 0 {
			errorResponse(w, http.StatusNotFound, "baseline not found or has no traces")
			return
		}

		target, err := database.GetTrace(traceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load trace")
			return
		}

		aggregate := graph.Aggregate(stepsByTrace(traces))
		result := graph.Overlay(aggregate, target.Steps)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
