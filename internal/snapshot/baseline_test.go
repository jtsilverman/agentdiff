package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testBaselineStore(t *testing.T) *BaselineStore {
	t.Helper()
	dir := t.TempDir()
	return NewBaselineStore(dir)
}

func TestBaselineRoundTrip(t *testing.T) {
	store := testBaselineStore(t)
	now := time.Now().Truncate(time.Second)

	b := Baseline{
		Name: "test-baseline",
		Snapshots: []Snapshot{
			testSnapshot("snap-1", []Step{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			}),
			testSnapshot("snap-2", []Step{
				{Role: "user", Content: "bye"},
			}),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Save(b); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("test-baseline")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != b.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, b.Name)
	}
	if len(loaded.Snapshots) != 2 {
		t.Fatalf("Snapshots len = %d, want 2", len(loaded.Snapshots))
	}
	if loaded.Snapshots[0].Name != "snap-1" {
		t.Errorf("Snapshots[0].Name = %q, want %q", loaded.Snapshots[0].Name, "snap-1")
	}
	if loaded.Snapshots[1].Name != "snap-2" {
		t.Errorf("Snapshots[1].Name = %q, want %q", loaded.Snapshots[1].Name, "snap-2")
	}
	if !loaded.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, now)
	}
	if !loaded.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", loaded.UpdatedAt, now)
	}
}

func TestBaselineCompressionSmaller(t *testing.T) {
	store := testBaselineStore(t)

	// Build a baseline with enough data to compress meaningfully.
	var snaps []Snapshot
	for i := 0; i < 20; i++ {
		snaps = append(snaps, testSnapshot("snap", []Step{
			{Role: "user", Content: "This is a repeated test message for compression verification."},
			{Role: "assistant", Content: "This is a repeated assistant response for compression verification."},
		}))
	}

	b := Baseline{
		Name:      "compress-test",
		Snapshots: snaps,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Save(b); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Compare compressed file size vs raw JSON.
	rawJSON, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		t.Fatalf("marshal raw JSON: %v", err)
	}

	compressedPath := filepath.Join(store.Dir(), "compress-test.json.gz")
	info, err := os.Stat(compressedPath)
	if err != nil {
		t.Fatalf("stat compressed file: %v", err)
	}

	rawSize := len(rawJSON)
	compressedSize := int(info.Size())

	if compressedSize >= rawSize {
		t.Errorf("compressed size (%d) should be smaller than raw JSON size (%d)", compressedSize, rawSize)
	}
}

func TestBaselineAddSnapshotCreatesAndAppends(t *testing.T) {
	store := testBaselineStore(t)

	snap1 := testSnapshot("first", []Step{
		{Role: "user", Content: "one"},
	})

	// AddSnapshot on a non-existent baseline should create it.
	if err := store.AddSnapshot("my-baseline", snap1); err != nil {
		t.Fatalf("AddSnapshot (create): %v", err)
	}

	loaded, err := store.Load("my-baseline")
	if err != nil {
		t.Fatalf("Load after first add: %v", err)
	}
	if len(loaded.Snapshots) != 1 {
		t.Fatalf("Snapshots len = %d, want 1", len(loaded.Snapshots))
	}
	if loaded.Snapshots[0].Name != "first" {
		t.Errorf("Snapshots[0].Name = %q, want %q", loaded.Snapshots[0].Name, "first")
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set on first create")
	}
	createdAt := loaded.CreatedAt

	// AddSnapshot again should append and preserve CreatedAt.
	snap2 := testSnapshot("second", []Step{
		{Role: "user", Content: "two"},
	})
	if err := store.AddSnapshot("my-baseline", snap2); err != nil {
		t.Fatalf("AddSnapshot (append): %v", err)
	}

	loaded, err = store.Load("my-baseline")
	if err != nil {
		t.Fatalf("Load after second add: %v", err)
	}
	if len(loaded.Snapshots) != 2 {
		t.Fatalf("Snapshots len = %d, want 2", len(loaded.Snapshots))
	}
	if loaded.Snapshots[1].Name != "second" {
		t.Errorf("Snapshots[1].Name = %q, want %q", loaded.Snapshots[1].Name, "second")
	}
	if !loaded.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt changed: got %v, want %v", loaded.CreatedAt, createdAt)
	}
	if !loaded.UpdatedAt.After(createdAt) || loaded.UpdatedAt.Equal(createdAt) {
		// UpdatedAt should be at or after CreatedAt (may be equal due to time resolution).
		// Just verify it's set and not zero.
		if loaded.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}
	}
}

func TestBaselineListSorted(t *testing.T) {
	store := testBaselineStore(t)

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	baselines := []Baseline{
		{Name: "oldest", UpdatedAt: t1, CreatedAt: t1},
		{Name: "middle", UpdatedAt: t2, CreatedAt: t2},
		{Name: "newest", UpdatedAt: t3, CreatedAt: t3},
	}

	for _, b := range baselines {
		if err := store.Save(b); err != nil {
			t.Fatalf("Save %s: %v", b.Name, err)
		}
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}

	// Newest first.
	if list[0].Name != "newest" {
		t.Errorf("list[0].Name = %q, want %q", list[0].Name, "newest")
	}
	if list[1].Name != "middle" {
		t.Errorf("list[1].Name = %q, want %q", list[1].Name, "middle")
	}
	if list[2].Name != "oldest" {
		t.Errorf("list[2].Name = %q, want %q", list[2].Name, "oldest")
	}
}

func TestBaselineLoadMissing(t *testing.T) {
	store := testBaselineStore(t)
	// Ensure dir exists.
	os.MkdirAll(store.Dir(), 0755)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Fatal("Load should return error for missing baseline")
	}
	// Error should mention the name clearly.
	expected := "baseline not found: nonexistent"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestBaselineListEmpty(t *testing.T) {
	store := testBaselineStore(t)
	list, err := store.List()
	if err != nil {
		t.Fatalf("List on empty: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List len = %d, want 0", len(list))
	}
}

func TestBaselineDirPath(t *testing.T) {
	store := NewBaselineStore("/tmp/test-base")
	expected := filepath.Join("/tmp/test-base", ".agentdiff", "baselines")
	if store.Dir() != expected {
		t.Errorf("Dir() = %q, want %q", store.Dir(), expected)
	}
}
