package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/cluster"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// toSnapshotBaseline converts DB trace details into a snapshot.Baseline for clustering.
func toSnapshotBaseline(name string, traces []db.TraceDetail) snapshot.Baseline {
	snaps := make([]snapshot.Snapshot, len(traces))
	for i, t := range traces {
		snaps[i] = snapshot.Snapshot{
			ID:        t.ID,
			Name:      t.Name,
			Source:    t.Source,
			Timestamp: t.CreatedAt,
			Metadata:  t.Metadata,
			Steps:     t.Steps,
		}
	}
	return snapshot.Baseline{
		Name:      name,
		Snapshots: snaps,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// GetCluster handles GET /api/baselines/:id/cluster.
func GetCluster(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		// Parse optional query params with defaults.
		epsilon := 0.0
		minPts := 2

		if epStr := r.URL.Query().Get("epsilon"); epStr != "" {
			ep, err := strconv.ParseFloat(epStr, 64)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, "invalid epsilon parameter")
				return
			}
			epsilon = ep
		}

		if mpStr := r.URL.Query().Get("min_points"); mpStr != "" {
			mp, err := strconv.Atoi(mpStr)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, "invalid min_points parameter")
				return
			}
			minPts = mp
		}

		traces, err := database.GetBaselineTraces(id)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to load baseline traces")
			return
		}

		if len(traces) == 0 {
			errorResponse(w, http.StatusNotFound, "baseline not found or has no traces")
			return
		}

		// Need to get baseline name. Use first trace query to get it from DB.
		baselines, err := database.ListBaselines()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to list baselines")
			return
		}

		var baselineName string
		for _, b := range baselines {
			if b.ID == id {
				baselineName = b.Name
				break
			}
		}

		baseline := toSnapshotBaseline(baselineName, traces)

		report, err := cluster.ClusterBaseline(baseline, epsilon, minPts)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "clustering failed: "+err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}
