package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir)
}

func testSnapshot(name string, steps []Step) Snapshot {
	return Snapshot{
		Name:      name,
		Source:    "test",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"env": "test"},
		Steps:     steps,
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	store := testStore(t)
	snap := testSnapshot("round-trip", []Step{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	})

	saved, err := store.Save(snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("Save should compute an ID")
	}

	loaded, err := store.Load("round-trip")
	if err != nil {
		t.Fatalf("Load by name: %v", err)
	}
	if loaded.ID != saved.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, saved.ID)
	}
	if loaded.Name != "round-trip" {
		t.Errorf("Name = %q, want %q", loaded.Name, "round-trip")
	}
	if len(loaded.Steps) != 2 {
		t.Errorf("Steps len = %d, want 2", len(loaded.Steps))
	}
}

func TestLoadByIDPrefix(t *testing.T) {
	store := testStore(t)
	snap := testSnapshot("id-test", []Step{
		{Role: "user", Content: "test content"},
	})

	saved, err := store.Save(snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load by first 6 chars of the ID.
	prefix := saved.ID[:6]
	loaded, err := store.Load(prefix)
	if err != nil {
		t.Fatalf("Load by ID prefix %q: %v", prefix, err)
	}
	if loaded.ID != saved.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, saved.ID)
	}
}

func TestLoadMissingSnapshot(t *testing.T) {
	store := testStore(t)
	// Ensure the directory exists so the scan doesn't fail on missing dir.
	os.MkdirAll(store.Dir(), 0755)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Fatal("Load should return error for missing snapshot")
	}
}

func TestListMultiple(t *testing.T) {
	store := testStore(t)

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	snaps := []Snapshot{
		{Name: "oldest", Source: "test", Timestamp: t1, Steps: []Step{{Role: "user", Content: "a"}}},
		{Name: "middle", Source: "test", Timestamp: t2, Steps: []Step{{Role: "user", Content: "b"}}},
		{Name: "newest", Source: "test", Timestamp: t3, Steps: []Step{{Role: "user", Content: "c"}}},
	}

	for _, s := range snaps {
		if _, err := store.Save(s); err != nil {
			t.Fatalf("Save %s: %v", s.Name, err)
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

func TestOverwriteOnDuplicate(t *testing.T) {
	store := testStore(t)

	snap1 := testSnapshot("overwrite-me", []Step{
		{Role: "user", Content: "version 1"},
	})
	saved1, err := store.Save(snap1)
	if err != nil {
		t.Fatalf("Save v1: %v", err)
	}

	snap2 := testSnapshot("overwrite-me", []Step{
		{Role: "user", Content: "version 2"},
	})
	saved2, err := store.Save(snap2)
	if err != nil {
		t.Fatalf("Save v2: %v", err)
	}

	if saved1.ID == saved2.ID {
		t.Error("different steps should produce different IDs")
	}

	loaded, err := store.Load("overwrite-me")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ID != saved2.ID {
		t.Errorf("loaded ID = %q, want %q (v2)", loaded.ID, saved2.ID)
	}
	if loaded.Steps[0].Content != "version 2" {
		t.Errorf("content = %q, want %q", loaded.Steps[0].Content, "version 2")
	}
}

func TestDirPath(t *testing.T) {
	store := NewStore("/tmp/test-base")
	expected := filepath.Join("/tmp/test-base", ".agentdiff", "snapshots")
	if store.Dir() != expected {
		t.Errorf("Dir() = %q, want %q", store.Dir(), expected)
	}
}

func TestSaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snap := testSnapshot("mkdir-test", []Step{
		{Role: "user", Content: "test"},
	})
	if _, err := store.Save(snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify directory was created.
	if _, err := os.Stat(store.Dir()); err != nil {
		t.Fatalf("snapshot dir should exist: %v", err)
	}

	// Verify file was created.
	path := filepath.Join(store.Dir(), "mkdir-test.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot file should exist: %v", err)
	}
}

func TestListEmptyDir(t *testing.T) {
	store := testStore(t)
	// Don't save anything; directory doesn't even exist yet.
	list, err := store.List()
	if err != nil {
		t.Fatalf("List on empty: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List len = %d, want 0", len(list))
	}
}

func TestSaveRequiresName(t *testing.T) {
	store := testStore(t)
	snap := Snapshot{Steps: []Step{{Role: "user", Content: "no name"}}}
	_, err := store.Save(snap)
	if err == nil {
		t.Fatal("Save should fail when name is empty")
	}
}
