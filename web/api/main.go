package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

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

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Printf("warning: ANTHROPIC_API_KEY not set; triage and transcripts will return deterministic fallbacks")
	}
	model := os.Getenv("ANTHROPIC_MODEL")
	triager := handlers.NewAnthropicTriager(apiKey, model)
	summarizer := handlers.NewAnthropicSummarizer(apiKey, model)

	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	RegisterRoutes(r, database, triager, summarizer)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("agentdiff-web listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
