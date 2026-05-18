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

// fakeTriager records call count and returns a canned TriageResult.
type fakeTriager struct {
	callCount int
	response  handlers.TriageResult
	err       error
}

func (f *fakeTriager) Triage(ctx context.Context, req handlers.TriageRequest) (handlers.TriageResult, error) {
	f.callCount++
	return f.response, f.err
}

// setupWithTriager mirrors setup() but registers the triage route with an injected Triager.
func setupWithTriager(t *testing.T, triager handlers.Triager) (*httptest.Server, *db.DB) {
	t.Helper()
	database, err := db.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory DB: %v", err)
	}
	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Route("/api", func(r chi.Router) {
		r.Post("/traces", handlers.PostTrace(database))
		r.Get("/traces", handlers.ListTraces(database))
		r.Get("/traces/{id}", handlers.GetTrace(database))
		r.Post("/baselines", handlers.PostBaseline(database))
		r.Get("/baselines", handlers.ListBaselines(database))
		r.Get("/baselines/{id}/cluster", handlers.GetCluster(database))
		r.Get("/diff/{idA}/{idB}", handlers.GetDiff(database))
		r.Post("/baselines/{id}/compare", handlers.PostCompare(database))
		r.Get("/baselines/{id}/graph", handlers.GetGraph(database))
		r.Get("/baselines/{id}/overlay/{traceID}", handlers.GetOverlay(database))
		r.Get("/diff/{idA}/{idB}/triage", handlers.GetTriage(database, triager))
	})
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		database.Close()
	})
	return ts, database
}

func TestGetTriageReturnsStructuredJSON(t *testing.T) {
	triager := &fakeTriager{
		response: handlers.TriageResult{
			Summary:        "trace B added a verification step before answering",
			Classification: "additive",
			LikelyCause:    "agent learned to verify before responding",
		},
	}
	ts, _ := setupWithTriager(t, triager)

	claudeData := readTestData(t, "claude_trace.jsonl")
	aResp := postTrace(t, ts, "trace-a", claudeData)
	bResp := postTrace(t, ts, "trace-b", claudeData)

	aID, _ := aResp["id"].(string)
	bID, _ := bResp["id"].(string)
	if aID == "" || bID == "" {
		t.Fatalf("missing trace IDs: aID=%q bID=%q", aID, bID)
	}

	url := fmt.Sprintf("%s/api/diff/%s/%s/triage", ts.URL, aID, bID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got["summary"] != "trace B added a verification step before answering" {
		t.Errorf("summary mismatch: got %v", got["summary"])
	}
	if got["classification"] != "additive" {
		t.Errorf("classification mismatch: got %v", got["classification"])
	}
	if got["likely_cause"] != "agent learned to verify before responding" {
		t.Errorf("likely_cause mismatch: got %v", got["likely_cause"])
	}
	if triager.callCount != 1 {
		t.Errorf("expected Triager called 1 time, got %d", triager.callCount)
	}
}

func TestGetTriageCachesResults(t *testing.T) {
	triager := &fakeTriager{
		response: handlers.TriageResult{
			Summary:        "cached result",
			Classification: "variance",
			LikelyCause:    "model nondeterminism",
		},
	}
	ts, _ := setupWithTriager(t, triager)

	claudeData := readTestData(t, "claude_trace.jsonl")
	aResp := postTrace(t, ts, "trace-a", claudeData)
	bResp := postTrace(t, ts, "trace-b", claudeData)
	aID, _ := aResp["id"].(string)
	bID, _ := bResp["id"].(string)
	if aID == "" || bID == "" {
		t.Fatalf("missing trace IDs: aID=%q bID=%q", aID, bID)
	}

	url := fmt.Sprintf("%s/api/diff/%s/%s/triage", ts.URL, aID, bID)

	// First call: cold cache, Triager invoked.
	resp1, err := http.Get(url)
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first call status %d", resp1.StatusCode)
	}
	if triager.callCount != 1 {
		t.Fatalf("expected callCount=1 after first call, got %d", triager.callCount)
	}

	// Second call: warm cache, Triager NOT invoked.
	start := time.Now()
	resp2, err := http.Get(url)
	if err != nil {
		t.Fatalf("second GET: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	elapsed := time.Since(start)

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second call status %d", resp2.StatusCode)
	}
	if triager.callCount != 1 {
		t.Errorf("expected callCount STILL 1 after cache hit, got %d", triager.callCount)
	}
	if !bytes.Equal(body1, body2) {
		t.Errorf("response body differs between cold and warm cache:\n  cold: %s\n  warm: %s", body1, body2)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("cache hit took %v, expected <100ms", elapsed)
	}
}
