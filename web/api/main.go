package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
	"github.com/jtsilverman/agentdiff/web/api/middleware"
	"github.com/jtsilverman/agentdiff/web/api/seed"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	dbPath := flag.String("db", "agentdiff.db", "SQLite database path")
	flag.Parse()

	database, err := db.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := seed.Seed(database); err != nil {
		log.Printf("warning: seed failed (continuing): %v", err)
	}
	if err := seed.LoadCaches(database); err != nil {
		log.Printf("warning: load caches failed (continuing): %v", err)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Printf("warning: ANTHROPIC_API_KEY not set; triage, transcripts, counterfactuals, and prompt-edit replays will return deterministic fallbacks")
	}
	model := os.Getenv("ANTHROPIC_MODEL")
	triager := handlers.NewAnthropicTriager(apiKey, model)
	summarizer := handlers.NewAnthropicSummarizer(apiKey, model)
	counterfactualer := handlers.NewAnthropicCounterfactual(apiKey, model)
	editPrompter := handlers.NewAnthropicEditPrompt(apiKey, model)

	voyageKey := os.Getenv("VOYAGE_API_KEY")
	if voyageKey == "" {
		log.Printf("warning: VOYAGE_API_KEY not set; trace embeddings will not be generated and /similar will return empty matches")
	}
	// NewVoyageEmbedder returns nil when the key is empty. Wrap as the
	// Embedder interface explicitly to avoid Go's typed-nil-interface gotcha
	// inside PostTrace's `embedder != nil` guard.
	var embedder handlers.Embedder
	if v := handlers.NewVoyageEmbedder(voyageKey, os.Getenv("VOYAGE_MODEL")); v != nil {
		embedder = v
	}

	killSwitch := os.Getenv("DEMO_KILL_SWITCH") == "true"
	if killSwitch {
		log.Printf("warning: DEMO_KILL_SWITCH=true — counterfactual + edit-prompt will always short-circuit to easter-egg payload (no LLM calls)")
	}
	rateLimit := middleware.RateLimit(middleware.RateLimitConfig{
		DB:         database,
		Threshold:  5,
		KillSwitch: killSwitch,
		Now:        time.Now,
	})

	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	RegisterRoutes(r, database, triager, summarizer, counterfactualer, editPrompter, embedder, rateLimit)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("agentdiff-web listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
