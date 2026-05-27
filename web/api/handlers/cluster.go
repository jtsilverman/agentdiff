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

// traceRef pairs a trace UUID with its human-readable name so the frontend
// can display the name on buttons but call downstream endpoints with the
// UUID. Trace names contain '/' (e.g. "seed-tool-order-stable/run-1") which
// breaks chi path routing when used as a URL segment.
// step_count + metadata are additive surfaces so baseline detail can render
// per-trace badges without a second listTraces fetch.
type traceRef struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	StepCount int               `json:"step_count,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type webStrategy struct {
	ID              int                       `json:"id"`
	Count           int                       `json:"count"`
	Exemplar        traceRef                  `json:"exemplar"`
	ToolSeq         []string                  `json:"tool_sequence"`
	Members         []traceRef                `json:"members"`
	MetadataSummary map[string]map[string]int `json:"metadata_summary,omitempty"`
}

type webStrategyReport struct {
	BaselineName  string        `json:"baseline_name"`
	SnapshotCount int           `json:"snapshot_count"`
	Strategies    []webStrategy `json:"strategies"`
	Noise         []traceRef    `json:"noise"`
	Epsilon       float64       `json:"epsilon"`
}

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

		traceByName := make(map[string]db.TraceDetail, len(traces))
		for _, t := range traces {
			traceByName[t.Name] = t
		}
		toRef := func(name string) traceRef {
			t, ok := traceByName[name]
			if !ok {
				return traceRef{Name: name}
			}
			return traceRef{
				ID:        t.ID,
				Name:      name,
				StepCount: len(t.Steps),
				Metadata:  t.Metadata,
			}
		}

		webReport := webStrategyReport{
			BaselineName:  report.BaselineName,
			SnapshotCount: report.SnapshotCount,
			Strategies:    make([]webStrategy, len(report.Strategies)),
			Noise:         make([]traceRef, len(report.Noise)),
			Epsilon:       report.Epsilon,
		}
		for i, s := range report.Strategies {
			members := make([]traceRef, len(s.Members))
			for j, m := range s.Members {
				members[j] = toRef(m)
			}
			webReport.Strategies[i] = webStrategy{
				ID:              s.ID,
				Count:           s.Count,
				Exemplar:        toRef(s.Exemplar),
				ToolSeq:         s.ToolSeq,
				Members:         members,
				MetadataSummary: s.MetadataSummary,
			}
		}
		for i, n := range report.Noise {
			webReport.Noise[i] = toRef(n)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(webReport)
	}
}
