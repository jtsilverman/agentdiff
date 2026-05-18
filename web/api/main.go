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

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Printf("warning: ANTHROPIC_API_KEY not set; triage will return deterministic fallbacks")
	}
	triager := handlers.NewAnthropicTriager(apiKey, os.Getenv("ANTHROPIC_MODEL"))

	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	RegisterRoutes(r, database, triager)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("agentdiff-web listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
