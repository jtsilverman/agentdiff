package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// sqlOpenLegacy opens a raw SQLite connection without running the schema
// migration — used to seed pre-chunk-18 fixtures the migration must upgrade.
func sqlOpenLegacy(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}

// snapshotColumnsViaPragma reads the snapshots table column set from PRAGMA
// table_info, the same query the migration uses internally.
func snapshotColumnsViaPragma(t *testing.T, db *DB) map[string]bool {
	t.Helper()
	rows, err := db.conn.Query("PRAGMA table_info(snapshots)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		cols[name] = true
	}
	return cols
}

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

// TestListTraces_BaselineMembership covers the chunk 7 additive fields:
// when a trace belongs to a baseline, ListTraces returns baseline_id +
// baseline_name; when a trace has no membership, both are empty strings.
func TestListTraces_BaselineMembership(t *testing.T) {
	db := testDB(t)

	tIn, _ := db.CreateTrace("in-baseline", "claude", nil)
	tOut, _ := db.CreateTrace("orphan", "claude", nil)
	bl, err := db.CreateBaseline("rate-limit", "Add rate limiting to a Flask endpoint", []string{tIn.ID})
	if err != nil {
		t.Fatalf("CreateBaseline: %v", err)
	}

	traces, err := db.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	byID := map[string]TraceSummary{}
	for _, tr := range traces {
		byID[tr.ID] = tr
	}
	if got := byID[tIn.ID]; got.BaselineID != bl.ID || got.BaselineName != "rate-limit" {
		t.Errorf("in-baseline trace: baseline_id=%q baseline_name=%q, want %q / %q",
			got.BaselineID, got.BaselineName, bl.ID, "rate-limit")
	}
	if got := byID[tOut.ID]; got.BaselineID != "" || got.BaselineName != "" {
		t.Errorf("orphan trace: baseline_id=%q baseline_name=%q, want empty",
			got.BaselineID, got.BaselineName)
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

func TestInsertSnapshots_RoundTripsCostAndLatency(t *testing.T) {
	db := testDB(t)

	tr, _ := db.CreateTrace("cost-trace", "claudecode", nil)
	cost1, lat1 := 532, 840
	cost2 := 588
	steps := []snapshot.Step{
		{Role: "assistant", Content: "thinking", CostTokens: &cost1, LatencyMs: &lat1},
		{
			Role: "tool_call",
			ToolCall: &snapshot.ToolCall{Name: "Read", Args: map[string]interface{}{"path": "x"}},
			// LatencyMs left nil — exercises mixed-nullability persistence.
			CostTokens: &cost2,
		},
		// Old-style step with no cost/latency.
		{Role: "assistant", Content: "done"},
	}
	if err := db.InsertSnapshots(tr.ID, steps); err != nil {
		t.Fatalf("InsertSnapshots: %v", err)
	}

	td, err := db.GetTrace(tr.ID)
	if err != nil {
		t.Fatalf("GetTrace: %v", err)
	}
	if len(td.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(td.Steps))
	}
	if td.Steps[0].CostTokens == nil || *td.Steps[0].CostTokens != 532 {
		t.Errorf("step 0: CostTokens want 532, got %v", td.Steps[0].CostTokens)
	}
	if td.Steps[0].LatencyMs == nil || *td.Steps[0].LatencyMs != 840 {
		t.Errorf("step 0: LatencyMs want 840, got %v", td.Steps[0].LatencyMs)
	}
	if td.Steps[1].CostTokens == nil || *td.Steps[1].CostTokens != 588 {
		t.Errorf("step 1: CostTokens want 588, got %v", td.Steps[1].CostTokens)
	}
	if td.Steps[1].LatencyMs != nil {
		t.Errorf("step 1: LatencyMs want nil, got %d", *td.Steps[1].LatencyMs)
	}
	if td.Steps[2].CostTokens != nil {
		t.Errorf("step 2: CostTokens want nil (legacy step), got %d", *td.Steps[2].CostTokens)
	}
}

func TestNewDB_MigratesExistingDBWithoutCostLatencyColumns(t *testing.T) {
	// Simulate a pre-chunk-18 DB by writing the old schema to a file, opening
	// it via NewDB (which must run the additive migration), then verifying both
	// columns exist and that legacy rows roundtrip with NULL cost/latency.
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")

	preChunk18Schema := `
CREATE TABLE traces (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    adapter     TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT '',
    metadata    TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE snapshots (
    id          TEXT PRIMARY KEY,
    trace_id    TEXT NOT NULL REFERENCES traces(id),
    step_index  INTEGER NOT NULL,
    role        TEXT NOT NULL,
    content     TEXT,
    tool_name   TEXT,
    tool_args   TEXT,
    tool_output TEXT,
    tool_is_error INTEGER NOT NULL DEFAULT 0
);
INSERT INTO traces (id, name, adapter) VALUES ('legacy-trace', 'old', 'claude');
INSERT INTO snapshots (id, trace_id, step_index, role, content)
    VALUES ('s1', 'legacy-trace', 0, 'user', 'hi');
`
	// Open with raw sqlite3 driver, apply the legacy schema, close.
	rawDB, err := sqlOpenLegacy(path)
	if err != nil {
		t.Fatalf("open legacy: %v", err)
	}
	if _, err := rawDB.Exec(preChunk18Schema); err != nil {
		rawDB.Close()
		t.Fatalf("exec legacy schema: %v", err)
	}
	rawDB.Close()

	// Now open via NewDB — migration must add cost_tokens + latency_ms.
	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("NewDB on legacy file: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cols := snapshotColumnsViaPragma(t, db)
	if !cols["cost_tokens"] {
		t.Errorf("migration: cost_tokens column not added")
	}
	if !cols["latency_ms"] {
		t.Errorf("migration: latency_ms column not added")
	}

	// Existing rows must read back with nil cost/latency, not crash.
	td, err := db.GetTrace("legacy-trace")
	if err != nil {
		t.Fatalf("GetTrace legacy: %v", err)
	}
	if len(td.Steps) != 1 {
		t.Fatalf("expected 1 legacy step, got %d", len(td.Steps))
	}
	if td.Steps[0].CostTokens != nil {
		t.Errorf("legacy step CostTokens want nil, got %d", *td.Steps[0].CostTokens)
	}
	if td.Steps[0].LatencyMs != nil {
		t.Errorf("legacy step LatencyMs want nil, got %d", *td.Steps[0].LatencyMs)
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

	bl, err := db.CreateBaseline("my-baseline", "", []string{tr1.ID, tr2.ID})
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
	db.CreateBaseline("dup-name", "", []string{tr.ID})

	_, err := db.CreateBaseline("dup-name", "", []string{tr.ID})
	if err == nil {
		t.Fatal("expected UNIQUE constraint error for duplicate baseline name")
	}
}

func TestCreateBaseline_InvalidTraceID(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateBaseline("bad-refs", "", []string{"nonexistent"})
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
	db.CreateBaseline("two-traces", "", []string{tr1.ID, tr2.ID})
	db.CreateBaseline("one-trace", "", []string{tr1.ID})

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

	bl, _ := db.CreateBaseline("with-steps", "", []string{tr.ID})

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

// --- Embedding CRUD tests ---

func TestInsertEmbedding_RoundTrip(t *testing.T) {
	db := testDB(t)
	tr, _ := db.CreateTrace("emb-trace", "claudecode", nil)

	vec := []float32{0.1, 0.2, 0.3, 0.4}
	if err := db.InsertEmbedding(tr.ID, vec, "voyage-3-lite"); err != nil {
		t.Fatalf("InsertEmbedding: %v", err)
	}

	got, err := db.GetEmbedding(tr.ID)
	if err != nil {
		t.Fatalf("GetEmbedding: %v", err)
	}
	if got.TraceID != tr.ID {
		t.Errorf("trace_id round-trip: expected %s, got %s", tr.ID, got.TraceID)
	}
	if got.ModelName != "voyage-3-lite" {
		t.Errorf("model_name round-trip: expected voyage-3-lite, got %s", got.ModelName)
	}
	if len(got.Vector) != 4 {
		t.Fatalf("vector length round-trip: expected 4, got %d", len(got.Vector))
	}
	for i, v := range vec {
		if got.Vector[i] != v {
			t.Errorf("vector[%d] round-trip: expected %v, got %v", i, v, got.Vector[i])
		}
	}
}

func TestGetEmbedding_NotFound(t *testing.T) {
	db := testDB(t)
	_, err := db.GetEmbedding("nonexistent-trace-id")
	if err == nil {
		t.Errorf("expected error on missing embedding, got nil")
	}
}

func TestListEmbeddings_ReturnsAll(t *testing.T) {
	db := testDB(t)
	t1, _ := db.CreateTrace("t1", "claudecode", nil)
	t2, _ := db.CreateTrace("t2", "claudecode", nil)
	t3, _ := db.CreateTrace("t3", "claudecode", nil) // no embedding for t3

	db.InsertEmbedding(t1.ID, []float32{1, 0}, "m1")
	db.InsertEmbedding(t2.ID, []float32{0, 1}, "m1")

	list, err := db.ListEmbeddings()
	if err != nil {
		t.Fatalf("ListEmbeddings: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 embeddings (t3 unembedded), got %d", len(list))
	}
	ids := map[string]bool{}
	for _, e := range list {
		ids[e.TraceID] = true
	}
	if !ids[t1.ID] || !ids[t2.ID] {
		t.Errorf("expected t1 + t2 in list, got %v", ids)
	}
	if ids[t3.ID] {
		t.Errorf("t3 (no embedding) leaked into list")
	}
}

func TestInsertEmbedding_UpsertOnConflict(t *testing.T) {
	// Inserting an embedding for the same trace twice should replace, not error
	// (trace_id is the PK; production embedder may re-run on the same trace).
	db := testDB(t)
	tr, _ := db.CreateTrace("upsert", "claudecode", nil)

	if err := db.InsertEmbedding(tr.ID, []float32{1, 0, 0}, "v1"); err != nil {
		t.Fatalf("first InsertEmbedding: %v", err)
	}
	if err := db.InsertEmbedding(tr.ID, []float32{0, 1, 0}, "v2"); err != nil {
		t.Fatalf("second InsertEmbedding (upsert): %v", err)
	}

	got, err := db.GetEmbedding(tr.ID)
	if err != nil {
		t.Fatalf("GetEmbedding after upsert: %v", err)
	}
	if got.ModelName != "v2" {
		t.Errorf("expected upserted model_name=v2, got %s", got.ModelName)
	}
	if got.Vector[1] != 1 {
		t.Errorf("expected upserted vector, got %v", got.Vector)
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
