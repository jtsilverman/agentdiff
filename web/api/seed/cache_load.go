package seed

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jtsilverman/agentdiff/web/api/db"
)

//go:embed seed-cache.json
var embeddedCacheJSON []byte

// LoadCaches reads the seed-cache.json baked into the binary at build time
// and inserts its entries via LoadCachesFromJSON. Called from main.go after
// Seed; safe on every boot (Put functions UPSERT, missing names skip per-entry).
func LoadCaches(database *db.DB) error {
	return LoadCachesFromJSON(database, embeddedCacheJSON)
}

type triageEntry struct {
	TraceAName     string `json:"trace_a_name"`
	TraceBName     string `json:"trace_b_name"`
	PromptsHash    string `json:"prompts_hash"`
	Summary        string `json:"summary"`
	Classification string `json:"classification"`
	LikelyCause    string `json:"likely_cause"`
}

type transcriptEntry struct {
	TraceName    string   `json:"trace_name"`
	PromptsHash  string   `json:"prompts_hash"`
	Summary      string   `json:"summary"`
	KeyDecisions []string `json:"key_decisions"`
}

type embeddingEntry struct {
	TraceName string    `json:"trace_name"`
	Vector    []float32 `json:"vector"`
	ModelName string    `json:"model_name"`
}

type cacheFile struct {
	Triage     []triageEntry     `json:"triage"`
	Transcript []transcriptEntry `json:"transcript"`
	Embeddings []embeddingEntry  `json:"embeddings"`
}

// LoadCachesFromJSON inserts pre-computed AI cache entries (triage, transcript,
// embeddings) into the existing DB cache tables, re-resolving stable trace
// names to freshly-generated trace IDs. Safe to call after Seed.
func LoadCachesFromJSON(database *db.DB, jsonBytes []byte) error {
	var cf cacheFile
	if err := json.Unmarshal(jsonBytes, &cf); err != nil {
		return fmt.Errorf("unmarshal cache JSON: %w", err)
	}

	nameToID, err := traceNameIndex(database)
	if err != nil {
		return err
	}

	var loaded, skipped int

	for _, t := range cf.Triage {
		aID, okA := nameToID[t.TraceAName]
		bID, okB := nameToID[t.TraceBName]
		if !okA || !okB {
			log.Printf("LoadCaches: SKIP triage entry %q/%q: trace name not found", t.TraceAName, t.TraceBName)
			skipped++
			continue
		}
		if err := database.PutTriageCache(aID, bID, t.PromptsHash, db.TriageCacheRow{
			Summary:        t.Summary,
			Classification: t.Classification,
			LikelyCause:    t.LikelyCause,
		}); err != nil {
			return fmt.Errorf("put triage cache for %s/%s: %w", t.TraceAName, t.TraceBName, err)
		}
		loaded++
	}

	for _, tr := range cf.Transcript {
		id, ok := nameToID[tr.TraceName]
		if !ok {
			log.Printf("LoadCaches: SKIP transcript entry %q: trace name not found", tr.TraceName)
			skipped++
			continue
		}
		if err := database.PutTranscriptCache(id, tr.PromptsHash, db.TranscriptCacheRow{
			Summary:      tr.Summary,
			KeyDecisions: tr.KeyDecisions,
		}); err != nil {
			return fmt.Errorf("put transcript cache for %s: %w", tr.TraceName, err)
		}
		loaded++
	}

	for _, e := range cf.Embeddings {
		id, ok := nameToID[e.TraceName]
		if !ok {
			log.Printf("LoadCaches: SKIP embedding entry %q: trace name not found", e.TraceName)
			skipped++
			continue
		}
		if err := database.InsertEmbedding(id, e.Vector, e.ModelName); err != nil {
			return fmt.Errorf("insert embedding for %s: %w", e.TraceName, err)
		}
		loaded++
	}

	log.Printf("LoadCaches: loaded %d entries, skipped %d (missing trace names)", loaded, skipped)
	return nil
}

func traceNameIndex(database *db.DB) (map[string]string, error) {
	traces, err := database.ListTraces()
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	out := make(map[string]string, len(traces))
	for _, t := range traces {
		out[t.Name] = t.ID
	}
	return out, nil
}
