package seed

import (
	"testing"

	"github.com/jtsilverman/agentdiff/web/api/db"
)

// TestLoadCachesFromJSON_RoundTripsTriageEntry is the acceptance hook for C5:
// a known triage entry in the cache JSON must be insertable via LoadCaches and
// readable via the existing handler-side GetTriageCache, with trace name → ID
// re-resolution against the freshly-seeded DB.
func TestLoadCachesFromJSON_RoundTripsTriageEntry(t *testing.T) {
	database := testDB(t)
	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	cacheJSON := []byte(`{
		"triage": [{
			"trace_a_name": "api-endpoint-rename/run-1",
			"trace_b_name": "api-endpoint-rename/run-2",
			"prompts_hash": "test-hash-abc",
			"summary": "Both runs follow the same read_file -> write_file sequence.",
			"classification": "variance",
			"likely_cause": "Deterministic execution path; no divergence."
		}],
		"transcript": [],
		"embeddings": []
	}`)

	if err := LoadCachesFromJSON(database, cacheJSON); err != nil {
		t.Fatalf("LoadCachesFromJSON: %v", err)
	}

	traceA := findTraceByName(t, database, "api-endpoint-rename/run-1")
	traceB := findTraceByName(t, database, "api-endpoint-rename/run-2")

	row, err := database.GetTriageCache(traceA.ID, traceB.ID, "test-hash-abc")
	if err != nil {
		t.Fatalf("GetTriageCache: %v", err)
	}
	if row.Classification != "variance" {
		t.Errorf("Classification: got %q, want %q", row.Classification, "variance")
	}
	if row.Summary != "Both runs follow the same read_file -> write_file sequence." {
		t.Errorf("Summary: got %q", row.Summary)
	}
	if row.LikelyCause != "Deterministic execution path; no divergence." {
		t.Errorf("LikelyCause: got %q", row.LikelyCause)
	}
}

// TestLoadCachesFromJSON_RoundTripsTranscriptEntry covers the transcript half
// of the round-trip: insert via LoadCaches, read via the existing handler-side
// GetTranscriptCache, including the JSON-encoded KeyDecisions slice.
func TestLoadCachesFromJSON_RoundTripsTranscriptEntry(t *testing.T) {
	database := testDB(t)
	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	cacheJSON := []byte(`{
		"triage": [],
		"transcript": [{
			"trace_name": "api-endpoint-rename/run-1",
			"prompts_hash": "transcript-hash-xyz",
			"summary": "Agent reads a file then writes back a transformed version.",
			"key_decisions": ["chose read_file before write_file", "no intermediate search needed"]
		}],
		"embeddings": []
	}`)

	if err := LoadCachesFromJSON(database, cacheJSON); err != nil {
		t.Fatalf("LoadCachesFromJSON: %v", err)
	}

	trace := findTraceByName(t, database, "api-endpoint-rename/run-1")

	row, err := database.GetTranscriptCache(trace.ID, "transcript-hash-xyz")
	if err != nil {
		t.Fatalf("GetTranscriptCache: %v", err)
	}
	if row.Summary != "Agent reads a file then writes back a transformed version." {
		t.Errorf("Summary: got %q", row.Summary)
	}
	if len(row.KeyDecisions) != 2 {
		t.Fatalf("KeyDecisions: got %d entries, want 2", len(row.KeyDecisions))
	}
	if row.KeyDecisions[0] != "chose read_file before write_file" {
		t.Errorf("KeyDecisions[0]: got %q", row.KeyDecisions[0])
	}
	if row.KeyDecisions[1] != "no intermediate search needed" {
		t.Errorf("KeyDecisions[1]: got %q", row.KeyDecisions[1])
	}
}

// TestLoadCachesFromJSON_RoundTripsEmbeddingEntry covers the embeddings half:
// vector + model_name keyed by stable trace name, readable via GetEmbedding
// against the freshly-generated trace ID.
func TestLoadCachesFromJSON_RoundTripsEmbeddingEntry(t *testing.T) {
	database := testDB(t)
	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	cacheJSON := []byte(`{
		"triage": [],
		"transcript": [],
		"embeddings": [{
			"trace_name": "api-endpoint-rename/run-1",
			"vector": [0.1, 0.2, 0.3, 0.4],
			"model_name": "voyage-3-lite"
		}]
	}`)

	if err := LoadCachesFromJSON(database, cacheJSON); err != nil {
		t.Fatalf("LoadCachesFromJSON: %v", err)
	}

	trace := findTraceByName(t, database, "api-endpoint-rename/run-1")

	emb, err := database.GetEmbedding(trace.ID)
	if err != nil {
		t.Fatalf("GetEmbedding: %v", err)
	}
	if emb.ModelName != "voyage-3-lite" {
		t.Errorf("ModelName: got %q, want %q", emb.ModelName, "voyage-3-lite")
	}
	wantVec := []float32{0.1, 0.2, 0.3, 0.4}
	if len(emb.Vector) != len(wantVec) {
		t.Fatalf("Vector length: got %d, want %d", len(emb.Vector), len(wantVec))
	}
	for i, v := range wantVec {
		if emb.Vector[i] != v {
			t.Errorf("Vector[%d]: got %v, want %v", i, emb.Vector[i], v)
		}
	}
}

// TestLoadCachesFromJSON_IdempotentOnReRun locks the Tier A guarantee that
// LoadCaches is safe to call on every boot. Re-running with the same JSON must
// not error and must leave the cache rows readable with the same payloads
// (relies on PutTriageCache / PutTranscriptCache INSERT OR REPLACE +
// InsertEmbedding ON CONFLICT DO UPDATE — locked behavior).
func TestLoadCachesFromJSON_IdempotentOnReRun(t *testing.T) {
	database := testDB(t)
	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	cacheJSON := []byte(`{
		"triage": [{
			"trace_a_name": "api-endpoint-rename/run-1",
			"trace_b_name": "api-endpoint-rename/run-2",
			"prompts_hash": "h",
			"summary": "s", "classification": "variance", "likely_cause": "lc"
		}],
		"transcript": [{
			"trace_name": "api-endpoint-rename/run-1",
			"prompts_hash": "h", "summary": "ts", "key_decisions": ["kd"]
		}],
		"embeddings": [{
			"trace_name": "api-endpoint-rename/run-1",
			"vector": [0.1, 0.2], "model_name": "voyage-3-lite"
		}]
	}`)

	if err := LoadCachesFromJSON(database, cacheJSON); err != nil {
		t.Fatalf("first LoadCachesFromJSON: %v", err)
	}
	if err := LoadCachesFromJSON(database, cacheJSON); err != nil {
		t.Fatalf("second LoadCachesFromJSON: %v", err)
	}

	traceA := findTraceByName(t, database, "api-endpoint-rename/run-1")
	traceB := findTraceByName(t, database, "api-endpoint-rename/run-2")

	if _, err := database.GetTriageCache(traceA.ID, traceB.ID, "h"); err != nil {
		t.Errorf("GetTriageCache after re-run: %v", err)
	}
	if _, err := database.GetTranscriptCache(traceA.ID, "h"); err != nil {
		t.Errorf("GetTranscriptCache after re-run: %v", err)
	}
	if _, err := database.GetEmbedding(traceA.ID); err != nil {
		t.Errorf("GetEmbedding after re-run: %v", err)
	}
}

// TestLoadCachesFromJSON_SkipsUnknownTraceNames locks the per-entry skip
// behavior on cache drift: a JSON entry referencing a renamed/missing trace
// name must not error the whole load. Other entries in the same JSON keep
// loading. Boot stays healthy; only the drifted entry falls back at request
// time.
func TestLoadCachesFromJSON_SkipsUnknownTraceNames(t *testing.T) {
	database := testDB(t)
	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	// One good triage entry alongside one entry referencing a renamed baseline.
	cacheJSON := []byte(`{
		"triage": [
			{
				"trace_a_name": "api-endpoint-rename/run-1",
				"trace_b_name": "api-endpoint-rename/run-2",
				"prompts_hash": "good-hash",
				"summary": "good", "classification": "variance", "likely_cause": "lc"
			},
			{
				"trace_a_name": "seed-renamed-old/run-1",
				"trace_b_name": "seed-renamed-old/run-2",
				"prompts_hash": "stale-hash",
				"summary": "stale", "classification": "regression", "likely_cause": "lc"
			}
		],
		"transcript": [],
		"embeddings": []
	}`)

	if err := LoadCachesFromJSON(database, cacheJSON); err != nil {
		t.Fatalf("LoadCachesFromJSON should not error on stale entry: %v", err)
	}

	traceA := findTraceByName(t, database, "api-endpoint-rename/run-1")
	traceB := findTraceByName(t, database, "api-endpoint-rename/run-2")

	row, err := database.GetTriageCache(traceA.ID, traceB.ID, "good-hash")
	if err != nil {
		t.Fatalf("good entry should still load when stale entry is present: %v", err)
	}
	if row.Summary != "good" {
		t.Errorf("good entry Summary: got %q, want %q", row.Summary, "good")
	}
}

func findTraceByName(t *testing.T, database *db.DB, name string) db.TraceSummary {
	t.Helper()
	traces, err := database.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	for _, tr := range traces {
		if tr.Name == name {
			return tr
		}
	}
	t.Fatalf("trace with name %q not found", name)
	return db.TraceSummary{}
}
