package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

type promoteRequest struct {
	Name string `json:"name"`
}

type promoteResponse struct {
	BaselineID   string `json:"baseline_id"`
	BaselineName string `json:"baseline_name"`
}

// PromoteTrace handles POST /api/traces/:id/promote.
// Creates a new baseline containing the single promoted trace and returns its id + name.
// Request body is optional: {"name": "..."}; if omitted, the baseline name is auto-generated.
func PromoteTrace(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := chi.URLParam(r, "id")

		// Verify the trace exists. 404 if not.
		trace, err := database.GetTrace(traceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to get trace")
			return
		}

		// Optional body. Empty body is allowed (auto-generated name).
		var req promoteRequest
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				errorResponse(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			short := trace.ID
			if len(short) > 8 {
				short = short[:8]
			}
			name = "promoted-" + short
		}

		baseline, err := database.CreateBaseline(name, "", []string{trace.ID})
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				errorResponse(w, http.StatusBadRequest, "baseline name already exists")
				return
			}
			if strings.Contains(err.Error(), "FOREIGN KEY") {
				errorResponse(w, http.StatusBadRequest, "invalid trace")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to create baseline")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(promoteResponse{
			BaselineID:   baseline.ID,
			BaselineName: baseline.Name,
		})
	}
}
