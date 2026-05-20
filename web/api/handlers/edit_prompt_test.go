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

// fakeEditPrompter records the request and returns a canned continuation.
type fakeEditPrompter struct {
	callCount int
	response  handlers.EditPromptResult
	err       error
	gotReq    handlers.EditPromptRequest
}

func (f *fakeEditPrompter) Replay(ctx context.Context, req handlers.EditPromptRequest) (handlers.EditPromptResult, error) {
	f.callCount++
	f.gotReq = req
	return f.response, f.err
}

func setupWithEditPrompter(t *testing.T, ep handlers.EditPrompter) (*httptest.Server, *db.DB) {
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
		r.Post("/traces/{id}/edit-prompt", handlers.PostEditPrompt(database, ep))
	})
	ts := httptest.NewServer(r)
	t.Cleanup(func() {
		ts.Close()
		database.Close()
	})
	return ts, database
}

func postEditPrompt(t *testing.T, tsURL, traceID string, body map[string]interface{}) (int, map[string]interface{}) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/api/traces/%s/edit-prompt", tsURL, traceID),
		"application/json",
		bytes.NewReader(raw),
	)
	if err != nil {
		t.Fatalf("POST edit-prompt: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestPostEditPrompt_HappyPath(t *testing.T) {
	ep := &fakeEditPrompter{
		response: handlers.EditPromptResult{
			NewSteps: []snapshot.Step{
				{Role: "assistant", Content: "Reframing: search source files only"},
				{Role: "tool", ToolCall: &snapshot.ToolCall{Name: "grep", Args: map[string]interface{}{"pattern": "TODO", "path": "src/"}}},
				{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "grep", Output: "src/main.go:42:// TODO", IsError: false}},
				{Role: "assistant", Content: "Found 1 TODO."},
			},
		},
	}
	ts, database := setupWithEditPrompter(t, ep)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "ep-source", claudeData)
	traceID := trace["id"].(string)

	status, body := postEditPrompt(t, ts.URL, traceID, map[string]interface{}{
		"step_index":      2,
		"modified_prompt": "Search only inside src/ this time.",
	})

	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%v)", status, body)
	}
	newTraceID, ok := body["new_trace_id"].(string)
	if !ok || newTraceID == "" {
		t.Fatalf("expected non-empty new_trace_id, got %v", body["new_trace_id"])
	}
	if newTraceID == traceID {
		t.Fatalf("edit-prompt new_trace_id must differ from original (%s)", traceID)
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
	if ep.callCount != 1 {
		t.Errorf("expected EditPrompter called 1 time, got %d", ep.callCount)
	}

	// FK row exists in prompt_edit_runs and links the new trace to the original.
	run, err := database.GetPromptEditRun(newTraceID)
	if err != nil {
		t.Fatalf("expected prompt_edit_runs row for %s, got error: %v", newTraceID, err)
	}
	if run.OriginalTraceID != traceID {
		t.Errorf("expected original_trace_id=%s, got %s", traceID, run.OriginalTraceID)
	}
	if run.StepIndex != 2 {
		t.Errorf("expected step_index=2, got %d", run.StepIndex)
	}
	if run.ModifiedPrompt != "Search only inside src/ this time." {
		t.Errorf("expected modified_prompt='Search only inside src/ this time.', got %q", run.ModifiedPrompt)
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

func TestPostEditPrompt_MissingTrace(t *testing.T) {
	ep := &fakeEditPrompter{}
	ts, _ := setupWithEditPrompter(t, ep)

	status, body := postEditPrompt(t, ts.URL, "00000000-0000-0000-0000-000000000000", map[string]interface{}{
		"step_index":      0,
		"modified_prompt": "x",
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%v)", status, body)
	}
	if ep.callCount != 0 {
		t.Errorf("expected EditPrompter NOT called on missing trace, got %d", ep.callCount)
	}
}

func TestPostEditPrompt_InvalidStepIndex(t *testing.T) {
	ep := &fakeEditPrompter{}
	ts, _ := setupWithEditPrompter(t, ep)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "ep-bad-step", claudeData)
	traceID := trace["id"].(string)

	// Negative step_index.
	status, _ := postEditPrompt(t, ts.URL, traceID, map[string]interface{}{
		"step_index":      -1,
		"modified_prompt": "x",
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 on negative step_index, got %d", status)
	}

	// step_index way past end of trace.
	status, _ = postEditPrompt(t, ts.URL, traceID, map[string]interface{}{
		"step_index":      9999,
		"modified_prompt": "x",
	})
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 on out-of-range step_index, got %d", status)
	}

	if ep.callCount != 0 {
		t.Errorf("expected EditPrompter NOT called on invalid step_index, got %d", ep.callCount)
	}
}

func TestPostEditPrompt_NewTraceContainsPrefixAndContinuation(t *testing.T) {
	// The new trace should be: original_steps[0:step_index] + modified-prompt step + new_steps.
	ep := &fakeEditPrompter{
		response: handlers.EditPromptResult{
			NewSteps: []snapshot.Step{
				{Role: "assistant", Content: "rewritten branch continuation"},
			},
		},
	}
	ts, _ := setupWithEditPrompter(t, ep)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "ep-shape", claudeData)
	traceID := trace["id"].(string)

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

	status, body := postEditPrompt(t, ts.URL, traceID, map[string]interface{}{
		"step_index":      2,
		"modified_prompt": "rewritten-prompt-text",
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

	// Shape: prefix (2 steps from original) + 1 modified-prompt step + 1 fake continuation = 4.
	wantLen := 2 + 1 + 1
	if len(newSteps) != wantLen {
		t.Errorf("expected %d total steps (prefix+modified+continuation), got %d", wantLen, len(newSteps))
	}

	// The modified-prompt step (at position 2) must contain the modified prompt text.
	if len(newSteps) > 2 {
		modStep := newSteps[2].(map[string]interface{})
		if modStep["content"] != "rewritten-prompt-text" {
			t.Errorf("expected modified step content='rewritten-prompt-text', got %v", modStep["content"])
		}
	}

	// The last step must be the fake continuation.
	if len(newSteps) > 0 {
		lastStep := newSteps[len(newSteps)-1].(map[string]interface{})
		if lastStep["content"] != "rewritten branch continuation" {
			t.Errorf("expected last step content='rewritten branch continuation', got %v", lastStep["content"])
		}
	}
}

func TestPostEditPrompt_EditPrompterError_FallsBackGracefully(t *testing.T) {
	// When the EditPrompter returns an error, the handler must still return 201
	// with a synthetic minimal trace recording the failure.
	ep := &fakeEditPrompter{
		err: errors.New("anthropic 503 service unavailable"),
	}
	ts, _ := setupWithEditPrompter(t, ep)

	claudeData := readTestData(t, "claude_trace.jsonl")
	trace := postTrace(t, ts, "ep-error-path", claudeData)
	traceID := trace["id"].(string)

	status, body := postEditPrompt(t, ts.URL, traceID, map[string]interface{}{
		"step_index":      1,
		"modified_prompt": "y",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 even on EditPrompter error, got %d (body=%v)", status, body)
	}
	if body["new_trace_id"] == nil || body["new_trace_id"] == "" {
		t.Errorf("expected new_trace_id even on fallback, got %v", body["new_trace_id"])
	}
}

// silence unused-import for sql.ErrNoRows when test fixtures change.
var _ = sql.ErrNoRows
