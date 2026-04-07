package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
	"github.com/jtsilverman/agentdiff/web/api/middleware"
)

// setup creates an in-memory DB, wires a Chi router, and returns an httptest.Server.
func setup(t *testing.T) (*httptest.Server, *db.DB) {
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
	})

	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		database.Close()
	})

	return ts, database
}

// readTestData reads a file from ../../testdata/ relative to this test file.
func readTestData(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/" + filename)
	if err != nil {
		t.Fatalf("failed to read test data %s: %v", filename, err)
	}
	return data
}

// postTrace is a helper that uploads a trace and returns the parsed response.
func postTrace(t *testing.T, ts *httptest.Server, name string, body []byte) map[string]interface{} {
	t.Helper()
	resp, err := http.Post(
		fmt.Sprintf("%s/api/traces?name=%s", ts.URL, name),
		"application/octet-stream",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST /api/traces failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return result
}

func TestPostAndGetTrace(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	// POST trace.
	result := postTrace(t, ts, "test-trace", claudeData)

	if result["adapter"] != "claude" {
		t.Errorf("expected adapter=claude, got %v", result["adapter"])
	}
	if result["name"] != "test-trace" {
		t.Errorf("expected name=test-trace, got %v", result["name"])
	}
	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected non-empty id, got %v", result["id"])
	}
	stepCount, ok := result["step_count"].(float64)
	if !ok || stepCount == 0 {
		t.Errorf("expected step_count > 0, got %v", result["step_count"])
	}

	// GET trace by ID.
	resp, err := http.Get(fmt.Sprintf("%s/api/traces/%s", ts.URL, id))
	if err != nil {
		t.Fatalf("GET /api/traces/%s failed: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var detail map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatalf("failed to decode trace detail: %v", err)
	}

	if detail["id"] != id {
		t.Errorf("expected id=%s, got %v", id, detail["id"])
	}
	steps, ok := detail["steps"].([]interface{})
	if !ok || len(steps) == 0 {
		t.Errorf("expected steps to be non-empty, got %v", detail["steps"])
	}
}

func TestListTraces(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	openaiData := readTestData(t, "openai_trace.json")

	// POST two traces with different adapters.
	r1 := postTrace(t, ts, "claude-trace", claudeData)
	r2 := postTrace(t, ts, "openai-trace", openaiData)

	if r1["adapter"] != "claude" {
		t.Errorf("expected adapter=claude, got %v", r1["adapter"])
	}
	if r2["adapter"] != "openai" {
		t.Errorf("expected adapter=openai, got %v", r2["adapter"])
	}

	// GET /api/traces.
	resp, err := http.Get(ts.URL + "/api/traces")
	if err != nil {
		t.Fatalf("GET /api/traces failed: %v", err)
	}
	defer resp.Body.Close()

	var traces []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&traces); err != nil {
		t.Fatalf("failed to decode traces list: %v", err)
	}

	if len(traces) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(traces))
	}

	// Verify both adapters are present.
	adapters := map[string]bool{}
	for _, tr := range traces {
		a, _ := tr["adapter"].(string)
		adapters[a] = true
	}
	if !adapters["claude"] || !adapters["openai"] {
		t.Errorf("expected both claude and openai adapters, got %v", adapters)
	}
}

func TestBaselineAndCluster(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	// Upload 3 identical traces (same tool sequence -> 1 cluster).
	var traceIDs []string
	for i := 0; i < 3; i++ {
		r := postTrace(t, ts, fmt.Sprintf("trace-%d", i), claudeData)
		id, _ := r["id"].(string)
		traceIDs = append(traceIDs, id)
	}

	// Create baseline.
	baselineBody, _ := json.Marshal(map[string]interface{}{
		"name":      "test-baseline",
		"trace_ids": traceIDs,
	})
	resp, err := http.Post(
		ts.URL+"/api/baselines",
		"application/json",
		bytes.NewReader(baselineBody),
	)
	if err != nil {
		t.Fatalf("POST /api/baselines failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for baseline, got %d", resp.StatusCode)
	}

	var baseline map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&baseline); err != nil {
		t.Fatalf("failed to decode baseline response: %v", err)
	}

	baselineID, _ := baseline["id"].(string)
	if baselineID == "" {
		t.Fatal("expected non-empty baseline ID")
	}

	traceCount, _ := baseline["trace_count"].(float64)
	if traceCount != 3 {
		t.Errorf("expected trace_count=3, got %v", traceCount)
	}

	// GET cluster.
	clusterResp, err := http.Get(fmt.Sprintf("%s/api/baselines/%s/cluster", ts.URL, baselineID))
	if err != nil {
		t.Fatalf("GET cluster failed: %v", err)
	}
	defer clusterResp.Body.Close()

	if clusterResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for cluster, got %d", clusterResp.StatusCode)
	}

	var report map[string]interface{}
	if err := json.NewDecoder(clusterResp.Body).Decode(&report); err != nil {
		t.Fatalf("failed to decode cluster report: %v", err)
	}

	strategies, ok := report["strategies"].([]interface{})
	if !ok || len(strategies) < 1 {
		t.Errorf("expected at least 1 strategy, got %v", report["strategies"])
	}
}

func TestDiff(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	openaiData := readTestData(t, "openai_trace.json")

	r1 := postTrace(t, ts, "claude-diff", claudeData)
	r2 := postTrace(t, ts, "openai-diff", openaiData)

	idA, _ := r1["id"].(string)
	idB, _ := r2["id"].(string)

	resp, err := http.Get(fmt.Sprintf("%s/api/diff/%s/%s", ts.URL, idA, idB))
	if err != nil {
		t.Fatalf("GET /api/diff failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var diffResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&diffResp); err != nil {
		t.Fatalf("failed to decode diff response: %v", err)
	}

	alignment, ok := diffResp["alignment"].([]interface{})
	if !ok || len(alignment) == 0 {
		t.Errorf("expected non-empty alignment, got %v", diffResp["alignment"])
	}

	distance, ok := diffResp["distance"].(float64)
	if !ok || distance <= 0 {
		t.Errorf("expected distance > 0, got %v", diffResp["distance"])
	}

	// Verify trace refs are populated.
	traceA, ok := diffResp["trace_a"].(map[string]interface{})
	if !ok || traceA["id"] != idA {
		t.Errorf("expected trace_a.id=%s, got %v", idA, traceA)
	}
	traceB, ok := diffResp["trace_b"].(map[string]interface{})
	if !ok || traceB["id"] != idB {
		t.Errorf("expected trace_b.id=%s, got %v", idB, traceB)
	}
}

func TestCompare(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	// Create 3 identical traces for baseline.
	var traceIDs []string
	for i := 0; i < 3; i++ {
		r := postTrace(t, ts, fmt.Sprintf("compare-trace-%d", i), claudeData)
		id, _ := r["id"].(string)
		traceIDs = append(traceIDs, id)
	}

	// Create baseline.
	baselineBody, _ := json.Marshal(map[string]interface{}{
		"name":      "compare-baseline",
		"trace_ids": traceIDs,
	})
	resp, err := http.Post(
		ts.URL+"/api/baselines",
		"application/json",
		bytes.NewReader(baselineBody),
	)
	if err != nil {
		t.Fatalf("POST /api/baselines failed: %v", err)
	}
	defer resp.Body.Close()

	var baseline map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&baseline)
	baselineID, _ := baseline["id"].(string)

	// Compare one of the same traces against the baseline.
	compareBody, _ := json.Marshal(map[string]string{
		"trace_id": traceIDs[0],
	})
	compareResp, err := http.Post(
		fmt.Sprintf("%s/api/baselines/%s/compare", ts.URL, baselineID),
		"application/json",
		bytes.NewReader(compareBody),
	)
	if err != nil {
		t.Fatalf("POST compare failed: %v", err)
	}
	defer compareResp.Body.Close()

	if compareResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", compareResp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(compareResp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode compare response: %v", err)
	}

	matched, ok := result["matched"].(bool)
	if !ok || !matched {
		t.Errorf("expected matched=true, got %v", result["matched"])
	}
}

func TestErrors(t *testing.T) {
	ts, _ := setup(t)

	t.Run("empty body returns 400", func(t *testing.T) {
		resp, err := http.Post(
			ts.URL+"/api/traces?name=empty",
			"application/octet-stream",
			bytes.NewReader([]byte{}),
		)
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("missing name param returns 400", func(t *testing.T) {
		resp, err := http.Post(
			ts.URL+"/api/traces",
			"application/octet-stream",
			bytes.NewReader([]byte("data")),
		)
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("nonexistent trace returns 404", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/traces/nonexistent-id-12345")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("baseline with empty name returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":      "",
			"trace_ids": []string{"some-id"},
		})
		resp, err := http.Post(
			ts.URL+"/api/baselines",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("baseline with no trace_ids returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":      "empty-baseline",
			"trace_ids": []string{},
		})
		resp, err := http.Post(
			ts.URL+"/api/baselines",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("diff with nonexistent trace returns 404", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/diff/fake-a/fake-b", ts.URL))
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("cluster with nonexistent baseline returns 404", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/baselines/fake-id/cluster", ts.URL))
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("cluster with invalid epsilon returns 400", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/baselines/any/cluster?epsilon=abc", ts.URL))
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("compare with missing trace_id returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"trace_id": ""})
		resp, err := http.Post(
			fmt.Sprintf("%s/api/baselines/any/compare", ts.URL),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("compare with nonexistent baseline returns 404", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"trace_id": "some-trace"})
		resp, err := http.Post(
			fmt.Sprintf("%s/api/baselines/fake-baseline/compare", ts.URL),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})
}
