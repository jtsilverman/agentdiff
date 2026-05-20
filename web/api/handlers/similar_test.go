package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
	"github.com/jtsilverman/agentdiff/web/api/middleware"
)

// fakeEmbedder returns canned vectors keyed on the trace ID it sees.
// This lets tests pre-arrange "similar / dissimilar" geometry deterministically
// without hitting the network or computing real embeddings.
type fakeEmbedder struct {
	vectors   map[string][]float32 // trace-name → canned vector
	callCount int
	err       error
}

func (f *fakeEmbedder) Embed(ctx context.Context, req handlers.EmbedRequest) (handlers.EmbedResult, error) {
	f.callCount++
	if f.err != nil {
		return handlers.EmbedResult{}, f.err
	}
	// Match by trace name (the test setup passes name through req).
	// Production matches by step content hash; tests use the trace-name shortcut for determinism.
	v, ok := f.vectors[req.TraceName]
	if !ok {
		// Default: return a "random-ish" vector so unknown traces still cluster separately.
		v = []float32{0.01, 0.01, 0.01, 0.01}
	}
	return handlers.EmbedResult{Vector: v, ModelName: "fake-embedder-v1"}, nil
}

// setupSimilar wires a server with POST /traces, GET /traces/{id}/similar, and
// a POST /test/embeddings/{id} helper for direct embedding insertion (sidesteps
// the trace-insert hook so geometry can be controlled per-test).
func setupSimilar(t *testing.T) (*httptest.Server, *db.DB) {
	t.Helper()
	database, err := db.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory DB: %v", err)
	}
	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Route("/api", func(r chi.Router) {
		r.Post("/traces", handlers.PostTrace(database, nil))
		r.Get("/traces/{id}", handlers.GetTrace(database))
		r.Get("/traces/{id}/similar", handlers.GetSimilar(database))
	})
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		database.Close()
	})
	return ts, database
}

// seedEmbedding writes a row into trace_embeddings directly via the DB API.
// Bypasses the embedder hook so test geometry is fully under test control.
func seedEmbedding(t *testing.T, database *db.DB, traceID string, vector []float32) {
	t.Helper()
	if err := database.InsertEmbedding(traceID, vector, "fake-embedder-v1"); err != nil {
		t.Fatalf("InsertEmbedding(%s): %v", traceID, err)
	}
}

func TestGetSimilar_HappyPath_RanksByCosine(t *testing.T) {
	ts, database := setupSimilar(t)

	// Insert 4 traces, embed them with vectors that produce a known ranking
	// against T1: T2 ≈ T1 (high), T3 partial overlap, T4 orthogonal.
	claudeData := readTestData(t, "claude_trace.jsonl")
	t1 := postTrace(t, ts, "sim-source", claudeData)["id"].(string)
	t2 := postTrace(t, ts, "sim-very-close", claudeData)["id"].(string)
	t3 := postTrace(t, ts, "sim-partial", claudeData)["id"].(string)
	t4 := postTrace(t, ts, "sim-orthogonal", claudeData)["id"].(string)

	seedEmbedding(t, database, t1, []float32{1, 1, 0, 0})
	seedEmbedding(t, database, t2, []float32{0.95, 1.05, 0.05, 0})
	seedEmbedding(t, database, t3, []float32{0.5, 0.5, 0.7, 0})
	seedEmbedding(t, database, t4, []float32{0, 0, 1, 1})

	resp, err := http.Get(fmt.Sprintf("%s/api/traces/%s/similar", ts.URL, t1))
	if err != nil {
		t.Fatalf("GET /similar: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	matches, ok := body["matches"].([]interface{})
	if !ok {
		t.Fatalf("expected matches array, got %v", body["matches"])
	}
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (T2, T3, T4 — source excluded), got %d", len(matches))
	}

	// Match shape: {trace_id, name, similarity_score}.
	first := matches[0].(map[string]interface{})
	if first["trace_id"] != t2 {
		t.Errorf("expected top match to be T2 (%s), got %s", t2, first["trace_id"])
	}
	if first["name"] == nil || first["name"] == "" {
		t.Errorf("expected name field, got %v", first["name"])
	}
	if _, ok := first["similarity_score"].(float64); !ok {
		t.Errorf("expected numeric similarity_score, got %v", first["similarity_score"])
	}

	// Source trace must not appear in its own results.
	for i, m := range matches {
		row := m.(map[string]interface{})
		if row["trace_id"] == t1 {
			t.Errorf("source trace (T1) appeared in results at position %d", i)
		}
	}

	// Sorted descending by similarity_score.
	prev := first["similarity_score"].(float64)
	for i := 1; i < len(matches); i++ {
		cur := matches[i].(map[string]interface{})["similarity_score"].(float64)
		if cur > prev+1e-6 {
			t.Errorf("matches not sorted descending: matches[%d].score=%v > matches[%d].score=%v",
				i, cur, i-1, prev)
		}
		prev = cur
	}
}

func TestGetSimilar_EmptyWhenSourceHasNoEmbedding(t *testing.T) {
	ts, database := setupSimilar(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	t1 := postTrace(t, ts, "no-emb", claudeData)["id"].(string)
	t2 := postTrace(t, ts, "has-emb", claudeData)["id"].(string)
	// Only T2 has an embedding; T1 does not.
	seedEmbedding(t, database, t2, []float32{1, 2, 3, 4})

	resp, _ := http.Get(fmt.Sprintf("%s/api/traces/%s/similar", ts.URL, t1))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (graceful empty), got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	matches := body["matches"].([]interface{})
	if len(matches) != 0 {
		t.Errorf("expected empty matches when source has no embedding, got %d", len(matches))
	}
}

func TestGetSimilar_ExcludesUnembeddedTraces(t *testing.T) {
	ts, database := setupSimilar(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	t1 := postTrace(t, ts, "src", claudeData)["id"].(string)
	t2 := postTrace(t, ts, "embedded", claudeData)["id"].(string)
	t3 := postTrace(t, ts, "no-embedding", claudeData)["id"].(string)
	seedEmbedding(t, database, t1, []float32{1, 1, 0, 0})
	seedEmbedding(t, database, t2, []float32{1, 1, 0, 0})
	// T3 intentionally has no embedding row.

	resp, _ := http.Get(fmt.Sprintf("%s/api/traces/%s/similar", ts.URL, t1))
	defer resp.Body.Close()
	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	matches := body["matches"].([]interface{})

	if len(matches) != 1 {
		t.Fatalf("expected 1 match (T2 only — T3 unembedded), got %d", len(matches))
	}
	if matches[0].(map[string]interface{})["trace_id"] != t2 {
		t.Errorf("expected only-match to be T2, got %v", matches[0])
	}
	for _, m := range matches {
		if m.(map[string]interface{})["trace_id"] == t3 {
			t.Errorf("T3 (unembedded) appeared in matches: %v", m)
		}
	}
}

func TestGetSimilar_TopFiveCap(t *testing.T) {
	ts, database := setupSimilar(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	src := postTrace(t, ts, "src", claudeData)["id"].(string)
	seedEmbedding(t, database, src, []float32{1, 0, 0, 0})

	// Insert 7 other traces, all with embeddings — handler should return only top 5.
	for i := 0; i < 7; i++ {
		id := postTrace(t, ts, fmt.Sprintf("other-%d", i), claudeData)["id"].(string)
		// Distinct vectors with decreasing similarity to src.
		v := []float32{float32(7-i) / 7.0, float32(i) / 7.0, 0.01, 0.01}
		seedEmbedding(t, database, id, v)
	}

	resp, _ := http.Get(fmt.Sprintf("%s/api/traces/%s/similar", ts.URL, src))
	defer resp.Body.Close()
	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	matches := body["matches"].([]interface{})

	if len(matches) != 5 {
		t.Errorf("expected exactly 5 matches (top-5 cap), got %d", len(matches))
	}
}

func TestGetSimilar_MissingTrace(t *testing.T) {
	ts, _ := setupSimilar(t)
	resp, _ := http.Get(fmt.Sprintf("%s/api/traces/%s/similar", ts.URL, "00000000-0000-0000-0000-000000000000"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 on missing trace, got %d", resp.StatusCode)
	}
}

func TestGetSimilar_SourceAloneEmbedded(t *testing.T) {
	// Source has an embedding; no other trace does. Result: empty matches, 200.
	ts, database := setupSimilar(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	src := postTrace(t, ts, "lonely", claudeData)["id"].(string)
	seedEmbedding(t, database, src, []float32{1, 0, 0, 0})

	resp, _ := http.Get(fmt.Sprintf("%s/api/traces/%s/similar", ts.URL, src))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	matches := body["matches"].([]interface{})
	if len(matches) != 0 {
		t.Errorf("expected empty matches when source alone is embedded, got %d", len(matches))
	}
}

// silence unused-import for snapshot if no fixture uses it.
var _ = snapshot.Step{}
