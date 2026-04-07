package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// testDB creates a temporary SQLite database and returns it with a cleanup function.
func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewDB(t *testing.T) {
	db := testDB(t)
	if db == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestNewDB_InvalidPath(t *testing.T) {
	_, err := NewDB("/nonexistent/dir/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestNewDB_SchemaIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db1, err := NewDB(path)
	if err != nil {
		t.Fatalf("first NewDB: %v", err)
	}
	db1.Close()

	// Opening again should not fail (CREATE IF NOT EXISTS).
	db2, err := NewDB(path)
	if err != nil {
		t.Fatalf("second NewDB: %v", err)
	}
	db2.Close()
}

// --- Trace tests ---

func TestCreateTrace(t *testing.T) {
	db := testDB(t)

	tr, err := db.CreateTrace("test-trace", "openai", map[string]string{"model": "gpt-4"})
	if err != nil {
		t.Fatalf("CreateTrace: %v", err)
	}
	if tr.ID == "" {
		t.Error("expected non-empty ID")
	}
	if tr.Name != "test-trace" {
		t.Errorf("name = %q, want %q", tr.Name, "test-trace")
	}
	if tr.Adapter != "openai" {
		t.Errorf("adapter = %q, want %q", tr.Adapter, "openai")
	}
	if tr.Metadata["model"] != "gpt-4" {
		t.Errorf("metadata[model] = %q, want %q", tr.Metadata["model"], "gpt-4")
	}
}

func TestCreateTrace_NilMetadata(t *testing.T) {
	db := testDB(t)

	tr, err := db.CreateTrace("no-meta", "claude", nil)
	if err != nil {
		t.Fatalf("CreateTrace: %v", err)
	}
	if tr.Metadata != nil {
		t.Errorf("expected nil metadata, got %v", tr.Metadata)
	}
}

func TestListTraces_Empty(t *testing.T) {
	db := testDB(t)

	traces, err := db.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(traces) != 0 {
		t.Errorf("expected 0 traces, got %d", len(traces))
	}
}

func TestListTraces_WithStepCount(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("with-steps", "openai", nil)
	steps := []snapshot.Step{
		{Role: "assistant", Content: "hello"},
		{Role: "user", Content: "hi"},
	}
	if err := db.InsertSnapshots(tr.ID, steps); err != nil {
		t.Fatalf("InsertSnapshots: %v", err)
	}

	traces, err := db.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].StepCount != 2 {
		t.Errorf("step_count = %d, want 2", traces[0].StepCount)
	}
}

func TestGetTrace_NotFound(t *testing.T) {
	db := testDB(t)

	_, err := db.GetTrace("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent trace")
	}
}

func TestGetTrace_WithSteps(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("full-trace", "claude", map[string]string{"version": "3"})
	steps := []snapshot.Step{
		{Role: "user", Content: "what's 2+2?"},
		{
			Role: "assistant",
			ToolCall: &snapshot.ToolCall{
				Name: "calculator",
				Args: map[string]interface{}{"expr": "2+2"},
			},
		},
		{
			Role: "tool",
			ToolResult: &snapshot.ToolResult{
				Name:   "calculator",
				Output: "4",
			},
		},
		{Role: "assistant", Content: "The answer is 4."},
	}
	if err := db.InsertSnapshots(tr.ID, steps); err != nil {
		t.Fatalf("InsertSnapshots: %v", err)
	}

	td, err := db.GetTrace(tr.ID)
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if td.Name != "full-trace" {
		t.Errorf("name = %q, want %q", td.Name, "full-trace")
	}
	if td.Metadata["version"] != "3" {
		t.Errorf("metadata[version] = %q, want %q", td.Metadata["version"], "3")
	}
	if len(td.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(td.Steps))
	}

	// Verify step order preserved.
	if td.Steps[0].Role != "user" || td.Steps[0].Content != "what's 2+2?" {
		t.Errorf("step 0: got role=%q content=%q", td.Steps[0].Role, td.Steps[0].Content)
	}

	// Verify tool call reconstruction.
	if td.Steps[1].ToolCall == nil {
		t.Fatal("step 1: expected ToolCall")
	}
	if td.Steps[1].ToolCall.Name != "calculator" {
		t.Errorf("step 1 tool name = %q, want %q", td.Steps[1].ToolCall.Name, "calculator")
	}

	// Verify tool result reconstruction.
	if td.Steps[2].ToolResult == nil {
		t.Fatal("step 2: expected ToolResult")
	}
	if td.Steps[2].ToolResult.Output != "4" {
		t.Errorf("step 2 output = %q, want %q", td.Steps[2].ToolResult.Output, "4")
	}
	if td.Steps[2].ToolResult.IsError {
		t.Error("step 2: expected IsError=false")
	}
}

func TestGetTrace_ToolResultError(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("error-trace", "openai", nil)
	steps := []snapshot.Step{
		{
			Role: "tool",
			ToolResult: &snapshot.ToolResult{
				Name:    "bash",
				Output:  "command not found",
				IsError: true,
			},
		},
	}
	db.InsertSnapshots(tr.ID, steps)

	td, err := db.GetTrace(tr.ID)
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if !td.Steps[0].ToolResult.IsError {
		t.Error("expected IsError=true")
	}
}

// --- Snapshot tests ---

func TestInsertSnapshots_Empty(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("empty-steps", "openai", nil)
	if err := db.InsertSnapshots(tr.ID, nil); err != nil {
		t.Fatalf("InsertSnapshots with nil: %v", err)
	}

	td, _ := db.GetTrace(tr.ID)
	if len(td.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(td.Steps))
	}
}

func TestInsertSnapshots_ForeignKeyViolation(t *testing.T) {
	db := testDB(t)

	steps := []snapshot.Step{{Role: "user", Content: "test"}}
	err := db.InsertSnapshots("nonexistent-trace", steps)
	if err == nil {
		t.Fatal("expected foreign key violation error")
	}
}

// --- Baseline tests ---

func TestCreateBaseline(t *testing.T) {
	db := testDB(t)

	tr1, _ := db.CreateTrace("trace-1", "openai", nil)
	tr2, _ := db.CreateTrace("trace-2", "claude", nil)

	bl, err := db.CreateBaseline("my-baseline", []string{tr1.ID, tr2.ID})
	if err != nil {
		t.Fatalf("CreateBaseline: %v", err)
	}
	if bl.ID == "" {
		t.Error("expected non-empty ID")
	}
	if bl.Name != "my-baseline" {
		t.Errorf("name = %q, want %q", bl.Name, "my-baseline")
	}
}

func TestCreateBaseline_DuplicateName(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("trace", "openai", nil)
	db.CreateBaseline("dup-name", []string{tr.ID})

	_, err := db.CreateBaseline("dup-name", []string{tr.ID})
	if err == nil {
		t.Fatal("expected UNIQUE constraint error for duplicate baseline name")
	}
}

func TestCreateBaseline_InvalidTraceID(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateBaseline("bad-refs", []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected foreign key violation error")
	}
}

func TestListBaselines_Empty(t *testing.T) {
	db := testDB(t)

	baselines, err := db.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines: %v", err)
	}
	if len(baselines) != 0 {
		t.Errorf("expected 0 baselines, got %d", len(baselines))
	}
}

func TestListBaselines_WithTraceCount(t *testing.T) {
	db := testDB(t)

	tr1, _ := db.CreateTrace("t1", "openai", nil)
	tr2, _ := db.CreateTrace("t2", "openai", nil)
	db.CreateBaseline("two-traces", []string{tr1.ID, tr2.ID})
	db.CreateBaseline("one-trace", []string{tr1.ID})

	baselines, err := db.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines: %v", err)
	}
	if len(baselines) != 2 {
		t.Fatalf("expected 2 baselines, got %d", len(baselines))
	}

	// Most recent first.
	counts := map[string]int{}
	for _, b := range baselines {
		counts[b.Name] = b.TraceCount
	}
	if counts["two-traces"] != 2 {
		t.Errorf("two-traces count = %d, want 2", counts["two-traces"])
	}
	if counts["one-trace"] != 1 {
		t.Errorf("one-trace count = %d, want 1", counts["one-trace"])
	}
}

func TestGetBaselineTraces(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("bl-trace", "claude", nil)
	steps := []snapshot.Step{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	db.InsertSnapshots(tr.ID, steps)

	bl, _ := db.CreateBaseline("with-steps", []string{tr.ID})

	details, err := db.GetBaselineTraces(bl.ID)
	if err != nil {
		t.Fatalf("GetBaselineTraces: %v", err)
	}
	if len(details) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(details))
	}
	if len(details[0].Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(details[0].Steps))
	}
}

func TestGetBaselineTraces_NonexistentBaseline(t *testing.T) {
	db := testDB(t)

	details, err := db.GetBaselineTraces("nonexistent")
	if err != nil {
		t.Fatalf("GetBaselineTraces: %v", err)
	}
	// Not an error, just empty.
	if len(details) != 0 {
		t.Errorf("expected 0 traces, got %d", len(details))
	}
}

// --- Close test ---

func TestClose(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Operations after close should fail.
	_, err = db.ListTraces()
	if err == nil {
		t.Fatal("expected error after Close")
	}
}

// Suppress unused import warning.
var _ = os.TempDir
