package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
	"github.com/jtsilverman/agentdiff/web/api/middleware"
)

// fakeSummarizer records call count and returns a canned TranscriptResult.
type fakeSummarizer struct {
	callCount int
	response  handlers.TranscriptResult
	err       error
}

func (f *fakeSummarizer) Summarize(ctx context.Context, req handlers.TranscriptRequest) (handlers.TranscriptResult, error) {
	f.callCount++
	return f.response, f.err
}

// setupWithSummarizer mirrors setup() but registers the transcript route with an injected Summarizer.
func setupWithSummarizer(t *testing.T, summarizer handlers.Summarizer) (*httptest.Server, *db.DB) {
	t.Helper()
	database, err := db.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory DB: %v", err)
	}
	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Route("/api", func(r chi.Router) {
		r.Post("/traces", handlers.PostTrace(database, nil))
		r.Get("/traces", handlers.ListTraces(database))
		r.Get("/traces/{id}", handlers.GetTrace(database))
		r.Get("/traces/{id}/transcript", handlers.GetTranscript(database, summarizer))
	})
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		database.Close()
	})
	return ts, database
}

func TestGetTranscriptReturnsStructuredJSON(t *testing.T) {
	summarizer := &fakeSummarizer{
		response: handlers.TranscriptResult{
			Summary:      "Agent listed the directory, read the largest file, and reported its line count.",
			KeyDecisions: []string{"chose ls over find", "read README.md first", "reported lines not bytes"},
		},
	}
	ts, _ := setupWithSummarizer(t, summarizer)

	claudeData := readTestData(t, "claude_trace.jsonl")
	resp := postTrace(t, ts, "trace-1", claudeData)
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatalf("missing trace ID")
	}

	url := fmt.Sprintf("%s/api/traces/%s/transcript", ts.URL, id)
	r, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", r.StatusCode)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["summary"] != "Agent listed the directory, read the largest file, and reported its line count." {
		t.Errorf("summary mismatch: got %v", got["summary"])
	}
	decisions, ok := got["key_decisions"].([]interface{})
	if !ok {
		t.Fatalf("key_decisions not a JSON array: got %T", got["key_decisions"])
	}
	if len(decisions) != 3 {
		t.Errorf("expected 3 key_decisions, got %d", len(decisions))
	}
	if summarizer.callCount != 1 {
		t.Errorf("expected Summarizer called 1 time, got %d", summarizer.callCount)
	}
}

func TestGetTranscriptCachesResults(t *testing.T) {
	summarizer := &fakeSummarizer{
		response: handlers.TranscriptResult{
			Summary:      "cached summary",
			KeyDecisions: []string{"first decision", "second decision"},
		},
	}
	ts, _ := setupWithSummarizer(t, summarizer)

	claudeData := readTestData(t, "claude_trace.jsonl")
	resp := postTrace(t, ts, "trace-1", claudeData)
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatalf("missing trace ID")
	}

	url := fmt.Sprintf("%s/api/traces/%s/transcript", ts.URL, id)

	// First call: cold cache, Summarizer invoked.
	r1, err := http.Get(url)
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}
	body1, _ := io.ReadAll(r1.Body)
	r1.Body.Close()
	if r1.StatusCode != http.StatusOK {
		t.Fatalf("first call status %d", r1.StatusCode)
	}
	if summarizer.callCount != 1 {
		t.Fatalf("expected callCount=1 after first call, got %d", summarizer.callCount)
	}

	// Second call: warm cache, Summarizer NOT invoked.
	start := time.Now()
	r2, err := http.Get(url)
	if err != nil {
		t.Fatalf("second GET: %v", err)
	}
	body2, _ := io.ReadAll(r2.Body)
	r2.Body.Close()
	elapsed := time.Since(start)

	if r2.StatusCode != http.StatusOK {
		t.Fatalf("second call status %d", r2.StatusCode)
	}
	if summarizer.callCount != 1 {
		t.Errorf("expected callCount STILL 1 after cache hit, got %d", summarizer.callCount)
	}
	if !bytes.Equal(body1, body2) {
		t.Errorf("response body differs between cold and warm cache:\n  cold: %s\n  warm: %s", body1, body2)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("cache hit took %v, expected <100ms", elapsed)
	}
}
