package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jtsilverman/agentdiff/web/api/db"
)

// baselineRequest is the JSON body for POST /api/baselines.
type baselineRequest struct {
	Name     string   `json:"name"`
	TraceIDs []string `json:"trace_ids"`
}

// baselineResponse is the JSON response for POST /api/baselines.
type baselineResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TraceCount int    `json:"trace_count"`
}

// baselineSummaryResponse is the JSON response for GET /api/baselines list items.
type baselineSummaryResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TraceCount int    `json:"trace_count"`
	CreatedAt  string `json:"created_at"`
}

// PostBaseline handles POST /api/baselines.
func PostBaseline(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req baselineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		defer r.Body.Close()

		if req.Name == "" {
			errorResponse(w, http.StatusBadRequest, "name is required")
			return
		}
		if len(req.TraceIDs) == 0 {
			errorResponse(w, http.StatusBadRequest, "at least one trace_id is required")
			return
		}

		baseline, err := database.CreateBaseline(req.Name, req.TraceIDs)
		if err != nil {
			if strings.Contains(err.Error(), "FOREIGN KEY") || strings.Contains(err.Error(), "UNIQUE") {
				errorResponse(w, http.StatusBadRequest, "invalid trace IDs or duplicate baseline name")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to create baseline")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(baselineResponse{
			ID:         baseline.ID,
			Name:       baseline.Name,
			TraceCount: len(req.TraceIDs),
		})
	}
}

// ListBaselines handles GET /api/baselines.
func ListBaselines(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		baselines, err := database.ListBaselines()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to list baselines")
			return
		}

		resp := make([]baselineSummaryResponse, len(baselines))
		for i, b := range baselines {
			resp[i] = baselineSummaryResponse{
				ID:         b.ID,
				Name:       b.Name,
				TraceCount: b.TraceCount,
				CreatedAt:  b.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
