package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/cluster"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// compareRequest is the JSON body for POST /api/baselines/{id}/compare.
type compareRequest struct {
	TraceID string `json:"trace_id"`
}

// PostCompare handles POST /api/baselines/{id}/compare.
func PostCompare(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		baselineID := chi.URLParam(r, "id")

		var req compareRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		defer r.Body.Close()

		if req.TraceID == "" {
			errorResponse(w, http.StatusBadRequest, "trace_id is required")
			return
		}

		// Load baseline traces.
		traces, err := database.GetBaselineTraces(baselineID)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to load baseline traces")
			return
		}
		if len(traces) == 0 {
			errorResponse(w, http.StatusNotFound, "baseline not found or has no traces")
			return
		}

		// Look up baseline name from the baselines table.
		var baselineName string
		baselines, err := database.ListBaselines()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to list baselines")
			return
		}
		for _, b := range baselines {
			if b.ID == baselineID {
				baselineName = b.Name
				break
			}
		}
		if baselineName == "" {
			baselineName = "baseline"
		}

		// Load comparison trace.
		trace, err := database.GetTrace(req.TraceID)
		if err != nil {
			if strings.Contains(err.Error(), "get trace") {
				errorResponse(w, http.StatusNotFound, "comparison trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to load comparison trace")
			return
		}

		// Convert to snapshot types using the package-level helper from cluster.go.
		baseline := toSnapshotBaseline(baselineName, traces)

		snap := snapshot.Snapshot{
			ID:        trace.ID,
			Name:      trace.Name,
			Source:    trace.Source,
			Timestamp: trace.CreatedAt,
			Metadata:  trace.Metadata,
			Steps:     trace.Steps,
		}

		result, err := cluster.CompareToCluster(baseline, snap, 0, 2)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "comparison failed: "+err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
