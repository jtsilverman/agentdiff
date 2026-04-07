package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
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

func TestPostTraceWithMetadata(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	metaJSON := `{"env":"prod"}`
	postURL := fmt.Sprintf("%s/api/traces?name=test&metadata=%s", ts.URL, url.QueryEscape(metaJSON))
	resp, err := http.Post(postURL, "application/octet-stream", bytes.NewReader(claudeData))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	id, _ := result["id"].(string)

	// GET the trace back and verify metadata.
	getResp, err := http.Get(fmt.Sprintf("%s/api/traces/%s", ts.URL, id))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer getResp.Body.Close()

	var detail map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&detail)

	metadata, ok := detail["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata map, got %v", detail["metadata"])
	}
	if metadata["env"] != "prod" {
		t.Errorf("expected metadata[env]=prod, got %v", metadata["env"])
	}
}

func TestPostTraceMetadataMerge(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	// The claude adapter detects model from the trace body.
	// User-provided metadata should override adapter-detected keys.
	metaJSON := `{"model":"custom-override"}`
	postURL := fmt.Sprintf("%s/api/traces?name=test&metadata=%s", ts.URL, url.QueryEscape(metaJSON))
	resp, err := http.Post(postURL, "application/octet-stream", bytes.NewReader(claudeData))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	id, _ := result["id"].(string)

	// GET the trace back and verify user metadata overrides adapter metadata.
	getResp, err := http.Get(fmt.Sprintf("%s/api/traces/%s", ts.URL, id))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer getResp.Body.Close()

	var detail map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&detail)

	metadata, ok := detail["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata map, got %v", detail["metadata"])
	}
	if metadata["model"] != "custom-override" {
		t.Errorf("expected metadata[model]=custom-override, got %v", metadata["model"])
	}
}

func TestPostTraceInvalidMetadata(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	postURL := fmt.Sprintf("%s/api/traces?name=test&metadata=%s", ts.URL, url.QueryEscape("notjson"))
	resp, err := http.Post(postURL, "application/octet-stream", bytes.NewReader(claudeData))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var errBody map[string]string
	json.NewDecoder(resp.Body).Decode(&errBody)
	if errMsg, ok := errBody["error"]; !ok || len(errMsg) == 0 {
		t.Errorf("expected error body with message, got %v", errBody)
	} else if !strings.Contains(errMsg, "invalid metadata") {
		t.Errorf("expected error to contain 'invalid metadata', got %q", errMsg)
	}
}


func TestListTracesIncludesMetadata(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	metaJSON := `{"env":"staging"}`
	postURL := fmt.Sprintf("%s/api/traces?name=meta-list-test&metadata=%s", ts.URL, url.QueryEscape(metaJSON))
	resp, err := http.Post(postURL, "application/octet-stream", bytes.NewReader(claudeData))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	// GET /api/traces and find our trace.
	listResp, err := http.Get(ts.URL + "/api/traces")
	if err != nil {
		t.Fatalf("GET /api/traces failed: %v", err)
	}
	defer listResp.Body.Close()

	var traces []map[string]interface{}
	json.NewDecoder(listResp.Body).Decode(&traces)

	found := false
	for _, tr := range traces {
		if tr["name"] == "meta-list-test" {
			found = true
			metadata, ok := tr["metadata"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected metadata map in list item, got %v", tr["metadata"])
			}
			if metadata["env"] != "staging" {
				t.Errorf("expected metadata[env]=staging, got %v", metadata["env"])
			}
			break
		}
	}
	if !found {
		t.Fatal("trace 'meta-list-test' not found in list response")
	}
}

func TestClusterMetadataSummary(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	// Create 3 traces with identical bodies (same tool sequence) but varying metadata.
	metas := []string{
		`{"model":"gpt-4"}`,
		`{"model":"gpt-4"}`,
		`{"model":"claude"}`,
	}

	var traceIDs []string
	for i, metaJSON := range metas {
		postURL := fmt.Sprintf("%s/api/traces?name=trace-%d&metadata=%s", ts.URL, i, url.QueryEscape(metaJSON))
		resp, err := http.Post(postURL, "application/octet-stream", bytes.NewReader(claudeData))
		if err != nil {
			t.Fatalf("POST trace-%d failed: %v", i, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("trace-%d: expected 201, got %d: %s", i, resp.StatusCode, body)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		id, _ := result["id"].(string)
		traceIDs = append(traceIDs, id)
	}

	// Create baseline with all 3 traces.
	baselineBody, _ := json.Marshal(map[string]interface{}{
		"name":      "meta-cluster-test",
		"trace_ids": traceIDs,
	})
	baseResp, err := http.Post(
		ts.URL+"/api/baselines",
		"application/json",
		bytes.NewReader(baselineBody),
	)
	if err != nil {
		t.Fatalf("POST /api/baselines failed: %v", err)
	}
	defer baseResp.Body.Close()

	if baseResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(baseResp.Body)
		t.Fatalf("expected 201 for baseline, got %d: %s", baseResp.StatusCode, body)
	}

	var baseline map[string]interface{}
	json.NewDecoder(baseResp.Body).Decode(&baseline)
	baselineID, _ := baseline["id"].(string)

	// GET cluster.
	clusterResp, err := http.Get(fmt.Sprintf("%s/api/baselines/%s/cluster", ts.URL, baselineID))
	if err != nil {
		t.Fatalf("GET cluster failed: %v", err)
	}
	defer clusterResp.Body.Close()

	if clusterResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(clusterResp.Body)
		t.Fatalf("expected 200 for cluster, got %d: %s", clusterResp.StatusCode, body)
	}

	var report map[string]interface{}
	json.NewDecoder(clusterResp.Body).Decode(&report)

	strategies, ok := report["strategies"].([]interface{})
	if !ok || len(strategies) == 0 {
		t.Fatalf("expected at least 1 strategy, got %v", report["strategies"])
	}

	// Find the strategy that contains all 3 members.
	var targetStrategy map[string]interface{}
	for _, s := range strategies {
		strat, _ := s.(map[string]interface{})
		members, _ := strat["members"].([]interface{})
		if len(members) == 3 {
			targetStrategy = strat
			break
		}
	}
	if targetStrategy == nil {
		t.Fatal("no strategy found with 3 members")
	}

	// Assert metadata_summary has model key with correct distribution.
	metaSummary, ok := targetStrategy["metadata_summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata_summary map, got %v", targetStrategy["metadata_summary"])
	}

	modelDist, ok := metaSummary["model"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata_summary[model] map, got %v", metaSummary["model"])
	}

	gpt4Count, _ := modelDist["gpt-4"].(float64)
	claudeCount, _ := modelDist["claude"].(float64)

	if gpt4Count != 2 {
		t.Errorf("expected gpt-4 count=2, got %v", gpt4Count)
	}
	if claudeCount != 1 {
		t.Errorf("expected claude count=1, got %v", claudeCount)
	}
}
