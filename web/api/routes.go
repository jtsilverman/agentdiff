package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
)

// RegisterRoutes wires all API handlers onto the router.
func RegisterRoutes(r chi.Router, database *db.DB) {
	r.Route("/api", func(r chi.Router) {
		// Trace endpoints
		r.Post("/traces", handlers.PostTrace(database))
		r.Get("/traces", handlers.ListTraces(database))
		r.Get("/traces/{id}", handlers.GetTrace(database))

		// Baseline endpoints
		r.Post("/baselines", handlers.PostBaseline(database))
		r.Get("/baselines", handlers.ListBaselines(database))
		r.Get("/baselines/{id}/cluster", handlers.GetCluster(database))

		// Diff and compare endpoints
		r.Get("/diff/{idA}/{idB}", handlers.GetDiff(database))
		r.Post("/baselines/{id}/compare", handlers.PostCompare(database))
	})
}
