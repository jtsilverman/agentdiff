package snapshot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Store manages snapshot persistence on disk.
type Store struct {
	dir string
}

// NewStore creates a Store rooted at baseDir/.agentdiff/snapshots/.
func NewStore(baseDir string) *Store {
	return &Store{
		dir: filepath.Join(baseDir, ".agentdiff", "snapshots"),
	}
}

// Dir returns the snapshots directory path.
func (s *Store) Dir() string {
	return s.dir
}

// computeID returns the first 12 hex chars of the SHA256 of the JSON-serialized Steps.
func computeID(steps []Step) (string, error) {
	data, err := json.Marshal(steps)
	if err != nil {
		return "", fmt.Errorf("marshal steps for ID: %w", err)
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])[:12], nil
}

// Save writes a snapshot to disk as <name>.json. It computes the ID from Steps,
// sets the timestamp if zero, and overwrites any existing file with the same name.
func (s *Store) Save(snap Snapshot) (Snapshot, error) {
	if snap.Name == "" {
		return Snapshot{}, fmt.Errorf("snapshot name is required")
	}

	id, err := computeID(snap.Steps)
	if err != nil {
		return Snapshot{}, err
	}
	snap.ID = id

	if snap.Timestamp.IsZero() {
		snap.Timestamp = time.Now()
	}

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return Snapshot{}, fmt.Errorf("create snapshot dir: %w", err)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal snapshot: %w", err)
	}

	path := filepath.Join(s.dir, snap.Name+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return Snapshot{}, fmt.Errorf("write snapshot: %w", err)
	}

	return snap, nil
}

// Load retrieves a snapshot by exact name or ID prefix.
// It tries exact name match first (<nameOrID>.json), then scans all files for an ID prefix match.
func (s *Store) Load(nameOrID string) (Snapshot, error) {
	// Try exact name match first.
	path := filepath.Join(s.dir, nameOrID+".json")
	if data, err := os.ReadFile(path); err == nil {
		var snap Snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			return Snapshot{}, fmt.Errorf("unmarshal snapshot %s: %w", nameOrID, err)
		}
		return snap, nil
	}

	// Scan for ID prefix match.
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, fmt.Errorf("snapshot not found: %s", nameOrID)
		}
		return Snapshot{}, fmt.Errorf("read snapshot dir: %w", err)
	}

	var matches []Snapshot
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			continue
		}
		if strings.HasPrefix(snap.ID, nameOrID) {
			matches = append(matches, snap)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return Snapshot{}, fmt.Errorf("ambiguous snapshot ID prefix %q: matches %d snapshots", nameOrID, len(matches))
	}

	return Snapshot{}, fmt.Errorf("snapshot not found: %s", nameOrID)
}

// List returns all snapshots sorted by timestamp (newest first).
func (s *Store) List() ([]Snapshot, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshot dir: %w", err)
	}

	var snapshots []Snapshot
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			continue
		}
		snapshots = append(snapshots, snap)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots, nil
}
