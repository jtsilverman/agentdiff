package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createBaseline uploads `count` copies of `traceData` and creates a named baseline.
// Returns baseline ID + uploaded trace IDs.
func createBaseline(t *testing.T, ts *httptest.Server, name string, traceData []byte, count int) (string, []string) {
	t.Helper()
	var traceIDs []string
	for i := 0; i < count; i++ {
		r := postTrace(t, ts, fmt.Sprintf("%s-trace-%d", name, i), traceData)
		id, _ := r["id"].(string)
		traceIDs = append(traceIDs, id)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"name":      name,
		"trace_ids": traceIDs,
	})
	resp, err := http.Post(ts.URL+"/api/baselines", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/baselines failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("baseline create: expected 201, got %d", resp.StatusCode)
	}
	var bl map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bl); err != nil {
		t.Fatalf("decode baseline: %v", err)
	}
	id, _ := bl["id"].(string)
	if id == "" {
		t.Fatal("empty baseline id")
	}
	return id, traceIDs
}

func TestGetGraph_HappyPath(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	// 3 identical traces -> nodes counted 3x, no branch points.
	baselineID, _ := createBaseline(t, ts, "graph-happy", claudeData, 3)

	resp, err := http.Get(fmt.Sprintf("%s/api/baselines/%s/graph", ts.URL, baselineID))
	if err != nil {
		t.Fatalf("GET graph failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode graph: %v", err)
	}

	nodes, ok := body["nodes"].([]interface{})
	if !ok {
		t.Fatalf("nodes missing or wrong type: %v", body["nodes"])
	}
	if len(nodes) == 0 {
		t.Error("expected at least one node")
	}
	if _, ok := body["edges"].([]interface{}); !ok {
		t.Errorf("edges missing or wrong type: %v", body["edges"])
	}
	stats, ok := body["stats"].(map[string]interface{})
	if !ok {
		t.Fatalf("stats missing or wrong type: %v", body["stats"])
	}
	if tr, _ := stats["total_runs"].(float64); tr != 3 {
		t.Errorf("stats.total_runs = %v, want 3", stats["total_runs"])
	}
	// Identical traces => no branch points.
	if bp, _ := stats["branch_points"].(float64); bp != 0 {
		t.Errorf("stats.branch_points = %v, want 0", stats["branch_points"])
	}

	// Verify node shape.
	firstNode, _ := nodes[0].(map[string]interface{})
	for _, k := range []string{"id", "tool_name", "count"} {
		if _, ok := firstNode[k]; !ok {
			t.Errorf("node missing key %q: %v", k, firstNode)
		}
	}
}

func TestGetGraph_MissingBaseline(t *testing.T) {
	ts, _ := setup(t)

	resp, err := http.Get(ts.URL + "/api/baselines/nonexistent/graph")
	if err != nil {
		t.Fatalf("GET graph failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetOverlay_HappyPath(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	baselineID, traceIDs := createBaseline(t, ts, "overlay-happy", claudeData, 3)

	// Overlay one of the baseline traces against itself -> all matched, no divergence.
	target := traceIDs[0]
	resp, err := http.Get(fmt.Sprintf("%s/api/baselines/%s/overlay/%s", ts.URL, baselineID, target))
	if err != nil {
		t.Fatalf("GET overlay failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode overlay: %v", err)
	}

	matchedNodes, ok := body["matched_node_ids"].([]interface{})
	if !ok {
		t.Fatalf("matched_node_ids missing: %v", body["matched_node_ids"])
	}
	if len(matchedNodes) == 0 {
		t.Error("expected at least one matched node")
	}
	if _, ok := body["matched_edge_ids"].([]interface{}); !ok {
		t.Errorf("matched_edge_ids missing: %v", body["matched_edge_ids"])
	}
	divergence, ok := body["divergence_points"].([]interface{})
	if !ok {
		t.Errorf("divergence_points missing: %v", body["divergence_points"])
	}
	// Trace was a baseline member; no divergence expected for the identical-trace case.
	if len(divergence) != 0 {
		t.Errorf("divergence_points = %v, want 0 for identical-trace overlay", divergence)
	}
}

func TestGetOverlay_MissingBaseline(t *testing.T) {
	ts, _ := setup(t)

	resp, err := http.Get(ts.URL + "/api/baselines/nope/overlay/whatever")
	if err != nil {
		t.Fatalf("GET overlay failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetOverlay_MissingTrace(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")

	baselineID, _ := createBaseline(t, ts, "overlay-missing-trace", claudeData, 3)

	resp, err := http.Get(fmt.Sprintf("%s/api/baselines/%s/overlay/nonexistent-trace", ts.URL, baselineID))
	if err != nil {
		t.Fatalf("GET overlay failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
