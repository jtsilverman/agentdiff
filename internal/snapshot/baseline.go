package snapshot

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Baseline groups multiple snapshots under a single name for statistical comparison.
type Baseline struct {
	Name      string     `json:"name"`
	Snapshots []Snapshot `json:"snapshots"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// BaselineStore manages baseline persistence on disk as gzipped JSON.
type BaselineStore struct {
	dir string
}

// NewBaselineStore creates a BaselineStore rooted at baseDir/.agentdiff/baselines/.
func NewBaselineStore(baseDir string) *BaselineStore {
	return &BaselineStore{
		dir: filepath.Join(baseDir, ".agentdiff", "baselines"),
	}
}

// Dir returns the baselines directory path.
func (bs *BaselineStore) Dir() string {
	return bs.dir
}

// Save marshals a Baseline to JSON, gzip-compresses it, and writes to <dir>/<name>.json.gz.
func (bs *BaselineStore) Save(b Baseline) error {
	if b.Name == "" {
		return fmt.Errorf("baseline name is required")
	}

	if err := os.MkdirAll(bs.dir, 0755); err != nil {
		return fmt.Errorf("create baseline dir: %w", err)
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}

	path := filepath.Join(bs.dir, b.Name+".json.gz")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create baseline file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	if _, err := gw.Write(data); err != nil {
		gw.Close()
		return fmt.Errorf("write gzip data: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	return nil
}

// Load reads a baseline by name from <dir>/<name>.json.gz.
// Returns a clear error if the file does not exist.
func (bs *BaselineStore) Load(name string) (Baseline, error) {
	path := filepath.Join(bs.dir, name+".json.gz")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Baseline{}, fmt.Errorf("baseline not found: %s", name)
		}
		return Baseline{}, fmt.Errorf("open baseline file: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return Baseline{}, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gr.Close()

	var b Baseline
	if err := json.NewDecoder(gr).Decode(&b); err != nil {
		return Baseline{}, fmt.Errorf("decode baseline: %w", err)
	}

	return b, nil
}

// List returns all baselines sorted by UpdatedAt (newest first).
func (bs *BaselineStore) List() ([]Baseline, error) {
	entries, err := os.ReadDir(bs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read baseline dir: %w", err)
	}

	var baselines []Baseline
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json.gz") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json.gz")
		b, err := bs.Load(name)
		if err != nil {
			continue
		}
		baselines = append(baselines, b)
	}

	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].UpdatedAt.After(baselines[j].UpdatedAt)
	})

	return baselines, nil
}

// AddSnapshot loads an existing baseline (or creates a new one), appends the snapshot,
// updates UpdatedAt, and saves. CreatedAt is set only on first creation.
func (bs *BaselineStore) AddSnapshot(name string, snap Snapshot) error {
	b, err := bs.Load(name)
	if err != nil {
		// Baseline does not exist; create a new one.
		now := time.Now()
		b = Baseline{
			Name:      name,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	b.Snapshots = append(b.Snapshots, snap)
	b.UpdatedAt = time.Now()

	return bs.Save(b)
}
