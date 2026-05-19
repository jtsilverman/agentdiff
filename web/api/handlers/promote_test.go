package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// postPromote is a helper that POSTs to /api/traces/:id/promote and returns (status, body).
func postPromote(t *testing.T, tsURL, traceID string, body map[string]string) (int, map[string]interface{}) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader([]byte{})
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/api/traces/%s/promote", tsURL, traceID),
		"application/json",
		reader,
	)
	if err != nil {
		t.Fatalf("POST promote failed: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestPromoteTrace_ExplicitName(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "promote-source-a", claudeData)
	traceID := trace["id"].(string)

	status, body := postPromote(t, ts.URL, traceID, map[string]string{"name": "my-promoted-baseline"})

	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%v)", status, body)
	}
	baselineID, ok := body["baseline_id"].(string)
	if !ok || baselineID == "" {
		t.Fatalf("expected non-empty baseline_id, got %v", body["baseline_id"])
	}
	if body["baseline_name"] != "my-promoted-baseline" {
		t.Errorf("expected baseline_name=my-promoted-baseline, got %v", body["baseline_name"])
	}

	// Verify the baseline appears in ListBaselines with trace_count == 1.
	resp, err := http.Get(ts.URL + "/api/baselines")
	if err != nil {
		t.Fatalf("GET baselines: %v", err)
	}
	defer resp.Body.Close()
	var baselines []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&baselines); err != nil {
		t.Fatalf("decode baselines: %v", err)
	}
	var found map[string]interface{}
	for _, b := range baselines {
		if b["id"] == baselineID {
			found = b
			break
		}
	}
	if found == nil {
		t.Fatalf("promoted baseline %s not in list", baselineID)
	}
	if count := found["trace_count"].(float64); count != 1 {
		t.Errorf("expected trace_count=1, got %v", count)
	}
}

func TestPromoteTrace_AutoName(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "auto-source", claudeData)
	traceID := trace["id"].(string)

	// No body at all → auto-generate name.
	status, body := postPromote(t, ts.URL, traceID, nil)

	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%v)", status, body)
	}
	name, ok := body["baseline_name"].(string)
	if !ok || name == "" {
		t.Fatalf("expected non-empty baseline_name, got %v", body["baseline_name"])
	}
	if !strings.HasPrefix(name, "promoted-") {
		t.Errorf("expected auto-name to start with 'promoted-', got %q", name)
	}
}

func TestPromoteTrace_MissingTrace(t *testing.T) {
	ts, _ := setup(t)

	status, body := postPromote(t, ts.URL, "00000000-0000-0000-0000-000000000000", map[string]string{"name": "x"})

	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%v)", status, body)
	}
}

func TestPromoteTrace_DuplicateName(t *testing.T) {
	ts, _ := setup(t)
	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "dup-source", claudeData)
	traceID := trace["id"].(string)

	// First promote succeeds.
	status, _ := postPromote(t, ts.URL, traceID, map[string]string{"name": "dup-baseline"})
	if status != http.StatusCreated {
		t.Fatalf("first promote: expected 201, got %d", status)
	}

	// Second promote with same explicit name collides.
	status, body := postPromote(t, ts.URL, traceID, map[string]string{"name": "dup-baseline"})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 on duplicate name, got %d (body=%v)", status, body)
	}
}
