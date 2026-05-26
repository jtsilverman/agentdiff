package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
)

// RegisterRoutes wires all API handlers onto the router. rateLimit wraps only
// the two expensive endpoints (counterfactual + edit-prompt) that generate
// fresh agent traces with no cache; all other routes pass through untouched.
func RegisterRoutes(r chi.Router, database *db.DB, triager handlers.Triager, summarizer handlers.Summarizer, counterfactualer handlers.Counterfactualer, editPrompter handlers.EditPrompter, embedder handlers.Embedder, rateLimit func(http.Handler) http.Handler) {
	r.Route("/api", func(r chi.Router) {
		// Trace endpoints
		r.Post("/traces", handlers.PostTrace(database, embedder))
		r.Get("/traces", handlers.ListTraces(database))
		r.Get("/traces/{id}", handlers.GetTrace(database))
		r.Get("/traces/{id}/transcript", handlers.GetTranscript(database, summarizer))
		r.Get("/traces/{id}/similar", handlers.GetSimilar(database))
		r.Post("/traces/{id}/promote", handlers.PromoteTrace(database))
		r.With(rateLimit).Post("/traces/{id}/counterfactual", handlers.PostCounterfactual(database, counterfactualer))
		r.With(rateLimit).Post("/traces/{id}/edit-prompt", handlers.PostEditPrompt(database, editPrompter))

		// Baseline endpoints
		r.Post("/baselines", handlers.PostBaseline(database))
		r.Get("/baselines", handlers.ListBaselines(database))
		r.Get("/baselines/{id}/cluster", handlers.GetCluster(database))

		// Diff and compare endpoints
		r.Get("/diff/{idA}/{idB}", handlers.GetDiff(database))
		r.Get("/diff/{idA}/{idB}/triage", handlers.GetTriage(database, triager))
		r.Post("/baselines/{id}/compare", handlers.PostCompare(database))

		// Path graph endpoints
		r.Get("/baselines/{id}/graph", handlers.GetGraph(database))
		r.Get("/baselines/{id}/overlay/{traceID}", handlers.GetOverlay(database))
	})
}
