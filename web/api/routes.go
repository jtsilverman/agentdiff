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

		// TODO: Baseline endpoints (Task 3)
		// r.Post("/baselines", handlers.CreateBaseline(database))
		// r.Get("/baselines", handlers.ListBaselines(database))
		// r.Get("/baselines/{id}", handlers.GetBaseline(database))

		// TODO: Cluster endpoints (Task 3)
		// r.Post("/traces/{id}/cluster", handlers.ClusterTrace(database))

		// TODO: Diff/Compare endpoints (Task 4)
		// r.Get("/diff", handlers.DiffTraces(database))
		// r.Get("/compare", handlers.CompareTraces(database))
	})
}
