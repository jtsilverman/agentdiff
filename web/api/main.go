package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
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

	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	RegisterRoutes(r, database)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("agentdiff-web listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
