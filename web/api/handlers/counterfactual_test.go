package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// fakeCounterfactualer records the request and returns a canned continuation.
type fakeCounterfactualer struct {
	callCount int
	response  handlers.CounterfactualResult
	err       error
	gotReq    handlers.CounterfactualRequest
}

func (f *fakeCounterfactualer) Replay(ctx context.Context, req handlers.CounterfactualRequest) (handlers.CounterfactualResult, error) {
	f.callCount++
	f.gotReq = req
	return f.response, f.err
}

// setupWithCounterfactualer wires a router with the counterfactual route + injected interface.
func setupWithCounterfactualer(t *testing.T, cf handlers.Counterfactualer) (*httptest.Server, *db.DB) {
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
		r.Post("/traces/{id}/counterfactual", handlers.PostCounterfactual(database, cf))
	})
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		database.Close()
	})
	return ts, database
}

func postCounterfactual(t *testing.T, tsURL, traceID string, body map[string]interface{}) (int, map[string]interface{}) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/api/traces/%s/counterfactual", tsURL, traceID),
		"application/json",
		bytes.NewReader(raw),
	)
	if err != nil {
		t.Fatalf("POST counterfactual: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestPostCounterfactual_HappyPath(t *testing.T) {
	cf := &fakeCounterfactualer{
		response: handlers.CounterfactualResult{
			NewSteps: []snapshot.Step{
				{Role: "assistant", Content: "Calling grep with new query"},
				{Role: "tool", ToolCall: &snapshot.ToolCall{Name: "grep", Args: map[string]interface{}{"pattern": "FIXME"}}},
				{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "grep", Output: "no matches", IsError: false}},
				{Role: "assistant", Content: "No FIXMEs found, done."},
			},
		},
	}
	ts, database := setupWithCounterfactualer(t, cf)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "cf-source", claudeData)
	traceID := trace["id"].(string)

	status, body := postCounterfactual(t, ts.URL, traceID, map[string]interface{}{
		"step_index":     2,
		"modified_input": "grep FIXME instead",
	})

	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%v)", status, body)
	}
	newTraceID, ok := body["new_trace_id"].(string)
	if !ok || newTraceID == "" {
		t.Fatalf("expected non-empty new_trace_id, got %v", body["new_trace_id"])
	}
	if newTraceID == traceID {
		t.Fatalf("counterfactual new_trace_id must differ from original (%s)", traceID)
	}
	comparison, ok := body["comparison"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected comparison object, got %v", body["comparison"])
	}
	if comparison["divergence_step"] == nil {
		t.Errorf("expected comparison.divergence_step, got nil")
	}
	if comparison["original_path"] == nil {
		t.Errorf("expected comparison.original_path, got nil")
	}
	if comparison["new_path"] == nil {
		t.Errorf("expected comparison.new_path, got nil")
	}
	if cf.callCount != 1 {
		t.Errorf("expected Counterfactualer called 1 time, got %d", cf.callCount)
	}

	// FK row exists in counterfactual_runs.
	run, err := database.GetCounterfactualRun(newTraceID)
	if err != nil {
		t.Fatalf("expected counterfactual_runs row for %s, got error: %v", newTraceID, err)
	}
	if run.OriginalTraceID != traceID {
		t.Errorf("expected original_trace_id=%s, got %s", traceID, run.OriginalTraceID)
	}
	if run.StepIndex != 2 {
		t.Errorf("expected step_index=2, got %d", run.StepIndex)
	}
	if run.ModifiedInput != "grep FIXME instead" {
		t.Errorf("expected modified_input='grep FIXME instead', got %q", run.ModifiedInput)
	}

	// The new trace must exist as a real trace with steps reconstructable.
	resp, err := http.Get(fmt.Sprintf("%s/api/traces/%s", ts.URL, newTraceID))
	if err != nil {
		t.Fatalf("GET new trace: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET new trace returned %d", resp.StatusCode)
	}
	var detail map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&detail)
	steps, ok := detail["steps"].([]interface{})
	if !ok || len(steps) == 0 {
		t.Errorf("expected new trace to have steps, got %v", detail["steps"])
	}
}

func TestPostCounterfactual_MissingTrace(t *testing.T) {
	cf := &fakeCounterfactualer{}
	ts, _ := setupWithCounterfactualer(t, cf)

	status, body := postCounterfactual(t, ts.URL, "00000000-0000-0000-0000-000000000000", map[string]interface{}{
		"step_index":     0,
		"modified_input": "x",
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%v)", status, body)
	}
	if cf.callCount != 0 {
		t.Errorf("expected Counterfactualer NOT called on missing trace, got %d", cf.callCount)
	}
}

func TestPostCounterfactual_InvalidStepIndex(t *testing.T) {
	cf := &fakeCounterfactualer{}
	ts, _ := setupWithCounterfactualer(t, cf)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "cf-bad-step", claudeData)
	traceID := trace["id"].(string)

	// Negative step_index.
	status, _ := postCounterfactual(t, ts.URL, traceID, map[string]interface{}{
		"step_index":     -1,
		"modified_input": "x",
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 on negative step_index, got %d", status)
	}

	// step_index way past end of trace.
	status, _ = postCounterfactual(t, ts.URL, traceID, map[string]interface{}{
		"step_index":     9999,
		"modified_input": "x",
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 on out-of-range step_index, got %d", status)
	}

	if cf.callCount != 0 {
		t.Errorf("expected Counterfactualer NOT called on invalid step_index, got %d", cf.callCount)
	}
}

func TestPostCounterfactual_NewTraceContainsPrefixAndContinuation(t *testing.T) {
	// The new trace should be: original_steps[0:step_index] + modified-input step + new_steps.
	// This ensures the counterfactual is a full, comparable trace (not just the forward suffix).
	cf := &fakeCounterfactualer{
		response: handlers.CounterfactualResult{
			NewSteps: []snapshot.Step{
				{Role: "assistant", Content: "branched continuation"},
			},
		},
	}
	ts, _ := setupWithCounterfactualer(t, cf)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "cf-shape", claudeData)
	traceID := trace["id"].(string)

	// Get original step count.
	resp, err := http.Get(fmt.Sprintf("%s/api/traces/%s", ts.URL, traceID))
	if err != nil {
		t.Fatalf("GET original: %v", err)
	}
	var origDetail map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&origDetail)
	resp.Body.Close()
	origSteps := origDetail["steps"].([]interface{})
	if len(origSteps) < 3 {
		t.Skipf("claude_trace.jsonl has %d steps; this test needs ≥3", len(origSteps))
	}

	status, body := postCounterfactual(t, ts.URL, traceID, map[string]interface{}{
		"step_index":     2,
		"modified_input": "branched-input-text",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%v)", status, body)
	}
	newTraceID := body["new_trace_id"].(string)

	resp, err = http.Get(fmt.Sprintf("%s/api/traces/%s", ts.URL, newTraceID))
	if err != nil {
		t.Fatalf("GET new trace: %v", err)
	}
	defer resp.Body.Close()
	var newDetail map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&newDetail)
	newSteps := newDetail["steps"].([]interface{})

	// Shape: prefix (2 steps from original) + 1 modified-input step + 1 fake continuation = 4.
	wantLen := 2 + 1 + 1
	if len(newSteps) != wantLen {
		t.Errorf("expected %d total steps (prefix+modified+continuation), got %d", wantLen, len(newSteps))
	}

	// The modified-input step (at position 2) must contain the modified input text.
	if len(newSteps) > 2 {
		modStep := newSteps[2].(map[string]interface{})
		if modStep["content"] != "branched-input-text" {
			t.Errorf("expected modified step content='branched-input-text', got %v", modStep["content"])
		}
	}

	// The last step must be the fake continuation.
	if len(newSteps) > 0 {
		lastStep := newSteps[len(newSteps)-1].(map[string]interface{})
		if lastStep["content"] != "branched continuation" {
			t.Errorf("expected last step content='branched continuation', got %v", lastStep["content"])
		}
	}
}

func TestPostCounterfactual_CounterfactualerError_FallsBackGracefully(t *testing.T) {
	// When the Counterfactualer returns an error, the handler must still return 201
	// with a synthetic minimal trace recording the failure — never bubble the error to the user.
	cf := &fakeCounterfactualer{
		err: errors.New("anthropic 503 service unavailable"),
	}
	ts, _ := setupWithCounterfactualer(t, cf)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "cf-error-path", claudeData)
	traceID := trace["id"].(string)

	status, body := postCounterfactual(t, ts.URL, traceID, map[string]interface{}{
		"step_index":     1,
		"modified_input": "y",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 even on Counterfactualer error, got %d (body=%v)", status, body)
	}
	if body["new_trace_id"] == nil || body["new_trace_id"] == "" {
		t.Errorf("expected new_trace_id even on fallback, got %v", body["new_trace_id"])
	}
}

// silence unused-import for sql.ErrNoRows when test fixtures change.
var _ = sql.ErrNoRows
