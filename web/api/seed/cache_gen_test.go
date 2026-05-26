//go:build genseed

package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
	"github.com/jtsilverman/agentdiff/web/api/handlers"
)

// TestGenerateSeedCache writes web/api/seed/seed-cache.json with hand-curated
// LLM content for the 5 seeded baselines. Triage + transcript portions only;
// embeddings are appended in a separate Voyage curl pass.
//
// Run with:
//
//	go test -tags genseed ./web/api/seed/ -run TestGenerateSeedCache
//
// Build-tag gated so normal `go test ./...` does not regenerate the file.
func TestGenerateSeedCache(t *testing.T) {
	database := testDB(t)
	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	stepsByName := loadStepsByName(t, database)

	cf := cacheFile{
		Triage:     buildTriageEntries(t, stepsByName),
		Transcript: buildTranscriptEntries(t, stepsByName),
		Embeddings: buildEmbeddingEntries(t, stepsByName),
	}

	out, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		t.Fatalf("marshal cache file: %v", err)
	}
	out = append(out, '\n')

	dest := filepath.Join("seed-cache.json")
	if err := os.WriteFile(dest, out, 0o644); err != nil {
		t.Fatalf("write %s: %v", dest, err)
	}
	t.Logf("wrote %d triage + %d transcript + %d embedding entries to %s", len(cf.Triage), len(cf.Transcript), len(cf.Embeddings), dest)
}

func loadStepsByName(t *testing.T, database *db.DB) map[string][]snapshot.Step {
	t.Helper()
	traces, err := database.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	out := make(map[string][]snapshot.Step, len(traces))
	for _, ts := range traces {
		td, err := database.GetTrace(ts.ID)
		if err != nil {
			t.Fatalf("GetTrace %s: %v", ts.Name, err)
		}
		out[ts.Name] = td.Steps
	}
	return out
}

type triagePair struct {
	a, b, summary, classification, likelyCause string
}

func buildTriageEntries(t *testing.T, steps map[string][]snapshot.Step) []triageEntry {
	t.Helper()

	pairs := []triagePair{
		// seed-tool-order-stable: all 5 runs identical → variance only.
		{
			a:              "seed-tool-order-stable/run-1",
			b:              "seed-tool-order-stable/run-2",
			summary:        "Both runs follow the same read_file → write_file sequence with no deviation. The agent's tool order is deterministic across both invocations.",
			classification: "variance",
			likelyCause:    "No behavioral change detected. Either both runs were guided by the same deterministic prompt path, or any prompt nondeterminism was resolved identically. This is the expected pattern for a stable agent on a well-specified task.",
		},

		// seed-tool-order-variance: short path (read,write) vs long path (search,read,write).
		{
			a:              "seed-tool-order-variance/run-1",
			b:              "seed-tool-order-variance/run-4",
			summary:        "Run-1 uses the direct read_file → write_file path; run-4 prepends a search step before reading. The longer path explores before committing to the read target, suggesting the agent treated the task as requiring discovery.",
			classification: "variance",
			likelyCause:    "Both paths are valid completions of the same task. The agent's choice between them likely depends on its interpretation of prompt phrasing (e.g., 'find and read X' vs 'read X'). This is healthy variance, not a regression, and reflects genuine ambiguity in the original prompt rather than a bug.",
		},
		{
			a:              "seed-tool-order-variance/run-3",
			b:              "seed-tool-order-variance/run-5",
			summary:        "Run-3 follows the short read_file → write_file path; run-5 follows the longer search → read_file → write_file path. The two are members of distinct behavioral strategies within this baseline.",
			classification: "variance",
			likelyCause:    "Cross-strategy comparison. Both strategies are present in this baseline (3 short runs, 2 long runs), so individual divergence between any short and long run is the expected outcome rather than a defect.",
		},

		// seed-prompt-regression: before-prompt-change vs after-prompt-change.
		{
			a:              "seed-prompt-regression/before-1",
			b:              "seed-prompt-regression/after-prompt-change",
			summary:        "Run before-1 includes an intermediate search step (read_file → search → write_file). The after-prompt-change run drops the search and goes straight from read_file to write_file. A behavior previously present is now absent.",
			classification: "regression",
			likelyCause:    "The most likely cause is a change to the system prompt or task description that no longer signals discovery is needed. If the search step was load-bearing (e.g., locating the correct file before reading), this regression is a silent quality drop. If search was incidental, the simpler path is an improvement. Recommend manually checking the new prompt against the original task requirements.",
		},
		{
			a:              "seed-prompt-regression/before-1",
			b:              "seed-prompt-regression/before-2",
			summary:        "Both runs follow the read_file → search → write_file sequence with identical tool selection and ordering. No divergence between baseline runs.",
			classification: "variance",
			likelyCause:    "Within-baseline reproducibility check. The agent behaves consistently across multiple invocations of the same pre-change prompt, so any divergence observed in the after-prompt-change run is attributable to the prompt change rather than to nondeterminism.",
		},

		// seed-novel-tool-discovery: standard vs novel-grep.
		{
			a:              "seed-novel-tool-discovery/run-1",
			b:              "seed-novel-tool-discovery/run-4",
			summary:        "Run-1 uses the canonical read_file → write_file path. Run-4 introduces a new grep step between read and write that does not appear in any earlier run.",
			classification: "additive",
			likelyCause:    "The agent has discovered (or been granted access to) the grep tool and is incorporating it into its workflow. This is additive behavior rather than a regression: the existing read+write capability is preserved, and a new search-within-content capability has been layered on top. Worth verifying that grep's output is being used downstream and not merely invoked.",
		},
		{
			a:              "seed-novel-tool-discovery/run-4",
			b:              "seed-novel-tool-discovery/run-5",
			summary:        "Both runs use the novel read_file → grep → write_file path consistently. The new tool sequence is stable across multiple invocations once the agent adopted it.",
			classification: "variance",
			likelyCause:    "Post-adoption stability check. The agent has settled on the grep-augmented workflow, suggesting either a stable prompt change or a learned-preference for the new pattern. No further divergence within the post-adoption cohort.",
		},

		// seed-noise-and-strategies: multiple strategies + outlier.
		{
			a:              "seed-noise-and-strategies/fetch-parse-store-1",
			b:              "seed-noise-and-strategies/fetch-store-1",
			summary:        "The fetch-parse-store run includes an intermediate parse step before storing; the fetch-store run goes directly from fetch to store, skipping parsing entirely.",
			classification: "variance",
			likelyCause:    "Two distinct strategies are present in this baseline. fetch-parse-store reflects a 'validate-then-persist' pattern; fetch-store reflects a 'store-raw-and-defer-parsing' pattern. Both are valid completions; the choice between them depends on whether the agent interprets the data as needing transformation at ingest time.",
		},
		{
			a:              "seed-noise-and-strategies/fetch-parse-store-1",
			b:              "seed-noise-and-strategies/outlier-bash-fetch",
			summary:        "The fetch-parse-store run uses three structured tools (fetch, parse, store). The outlier run invokes bash before fetch and omits parse and store entirely, producing a fundamentally different shape.",
			classification: "regression",
			likelyCause:    "The outlier run is incomplete by the standard of every other baseline trace: no persistence step, prefixed by an ad-hoc bash invocation. This pattern suggests either a degraded prompt that failed to elicit the structured workflow, or a model-side regression that drops the structured-output guarantee. Strong candidate for triage as a real defect, not benign variance.",
		},
		{
			a:              "seed-noise-and-strategies/fetch-store-1",
			b:              "seed-noise-and-strategies/outlier-bash-fetch",
			summary:        "The fetch-store run completes the persistence step; the outlier run replaces it with bash. Both omit parsing, but only one persists.",
			classification: "regression",
			likelyCause:    "Even compared against the more permissive fetch-store strategy, the outlier drops the persistence step. The bash invocation is unstructured and not wired to downstream consumers, so the run effectively produced no usable output. This is a clear regression against the demonstrated minimum-viable path.",
		},
	}

	out := make([]triageEntry, 0, len(pairs))
	for _, p := range pairs {
		stepsA, ok := steps[p.a]
		if !ok {
			t.Fatalf("triage pair: trace %q not in seeded DB", p.a)
		}
		stepsB, ok := steps[p.b]
		if !ok {
			t.Fatalf("triage pair: trace %q not in seeded DB", p.b)
		}
		out = append(out, triageEntry{
			TraceAName:     p.a,
			TraceBName:     p.b,
			PromptsHash:    handlers.TriagePromptsHash(stepsA, stepsB),
			Summary:        p.summary,
			Classification: p.classification,
			LikelyCause:    p.likelyCause,
		})
	}
	return out
}

type transcriptSpec struct {
	traceName    string
	summary      string
	keyDecisions []string
}

func buildTranscriptEntries(t *testing.T, steps map[string][]snapshot.Step) []transcriptEntry {
	t.Helper()

	// Templates by content shape. Each baseline's traces fall into one of these
	// patterns; reusing template text keeps the demo readable without claiming
	// per-run insight the model wouldn't actually have.
	readWriteSummary := "The agent received a prompt to read and write files, then executed the task in two tool calls: read_file followed by write_file. No intermediate exploration or branching was required; the task's structure was clear enough to commit directly to the canonical two-step path."
	readWriteDecisions := []string{
		"Chose to read before writing, establishing the input baseline before producing output.",
		"Skipped exploratory tools (search, grep) because the prompt did not require discovering content.",
	}

	searchReadWriteSummary := "The agent took a discovery-first path: it searched before reading, then wrote. The presence of the search step suggests the agent interpreted the task as requiring it to locate the target before consuming it, rather than receiving the target reference directly in the prompt."
	searchReadWriteDecisions := []string{
		"Invoked search before read_file to locate the target rather than assuming the prompt named it directly.",
		"Followed the search with a single read_file call rather than chaining multiple reads, indicating the search returned a precise enough result.",
		"Closed with write_file to commit the output, matching the prompt's expected deliverable.",
	}

	readSearchWriteSummary := "The agent executed read_file → search → write_file. Reading first established the input baseline; the search step likely inspected content for a specific pattern before writing the final output. This shape matches a 'load then query' pattern rather than a pure transform."
	readSearchWriteDecisions := []string{
		"Loaded the file before searching, suggesting the search operated on the file's contents rather than on the filesystem.",
		"Used a single search-and-write cycle rather than iterative refinement, indicating the search returned an actionable result on the first pass.",
	}

	readGrepWriteSummary := "The agent invoked grep between read_file and write_file. The grep step filters or extracts content from the loaded file, signaling that the agent treated the task as requiring pattern-based extraction rather than wholesale transformation."
	readGrepWriteDecisions := []string{
		"Adopted grep as an intermediate filter rather than asking the model to do the filtering inline.",
		"Preserved the read-then-write skeleton, layering grep on top rather than replacing existing steps.",
	}

	fetchParseStoreSummary := "The agent followed a three-stage ingestion pipeline: fetch the data, parse it into a structured form, then store the parsed result. This is the canonical 'validate at ingest' pattern that catches malformed inputs at parse time rather than at downstream consumption."
	fetchParseStoreDecisions := []string{
		"Inserted a parse step between fetch and store rather than persisting raw input.",
		"Treated parsing as a contract enforcement point: invalid input fails here, not later.",
	}

	fetchStoreSummary := "The agent skipped parsing and stored the raw fetched payload directly. This 'store-raw-and-defer-parsing' pattern trades early validation for write throughput; downstream consumers must handle parsing themselves."
	fetchStoreDecisions := []string{
		"Omitted parse, suggesting the agent interpreted the storage layer as schema-flexible.",
		"Reduced the pipeline to two stages, minimizing per-record processing cost at write time.",
	}

	outlierBashFetchSummary := "The agent invoked bash first, then fetch, then stopped. No parse step, no store step. The bash prefix is unusual: most agents in this baseline use structured tools end-to-end rather than dropping to shell. The run terminated without producing a persisted artifact, suggesting either a workflow breakdown or a deliberate exploratory deviation."
	outlierBashFetchDecisions := []string{
		"Reached for bash before any structured tool, breaking from the baseline's structured-tool-first convention.",
		"Stopped at fetch without invoking store, leaving the fetched data unpersisted.",
		"Omitted parsing entirely, consistent with the truncated pipeline.",
	}

	specs := []transcriptSpec{
		// seed-tool-order-stable — all 5 traces identical (read_file, write_file).
		{traceName: "seed-tool-order-stable/run-1", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-stable/run-2", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-stable/run-3", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-stable/run-4", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-stable/run-5", summary: readWriteSummary, keyDecisions: readWriteDecisions},

		// seed-tool-order-variance — runs 1-3 read+write, runs 4-5 search+read+write.
		{traceName: "seed-tool-order-variance/run-1", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-variance/run-2", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-variance/run-3", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-tool-order-variance/run-4", summary: searchReadWriteSummary, keyDecisions: searchReadWriteDecisions},
		{traceName: "seed-tool-order-variance/run-5", summary: searchReadWriteSummary, keyDecisions: searchReadWriteDecisions},

		// seed-prompt-regression — before-1..4 read+search+write, after-prompt-change read+write.
		{traceName: "seed-prompt-regression/before-1", summary: readSearchWriteSummary, keyDecisions: readSearchWriteDecisions},
		{traceName: "seed-prompt-regression/before-2", summary: readSearchWriteSummary, keyDecisions: readSearchWriteDecisions},
		{traceName: "seed-prompt-regression/before-3", summary: readSearchWriteSummary, keyDecisions: readSearchWriteDecisions},
		{traceName: "seed-prompt-regression/before-4", summary: readSearchWriteSummary, keyDecisions: readSearchWriteDecisions},
		{traceName: "seed-prompt-regression/after-prompt-change", summary: readWriteSummary, keyDecisions: readWriteDecisions},

		// seed-novel-tool-discovery — runs 1-3 read+write, runs 4-5 read+grep+write.
		{traceName: "seed-novel-tool-discovery/run-1", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-novel-tool-discovery/run-2", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-novel-tool-discovery/run-3", summary: readWriteSummary, keyDecisions: readWriteDecisions},
		{traceName: "seed-novel-tool-discovery/run-4", summary: readGrepWriteSummary, keyDecisions: readGrepWriteDecisions},
		{traceName: "seed-novel-tool-discovery/run-5", summary: readGrepWriteSummary, keyDecisions: readGrepWriteDecisions},

		// seed-noise-and-strategies — 3× fetch-parse-store, 2× fetch-store, 1× outlier.
		{traceName: "seed-noise-and-strategies/fetch-parse-store-1", summary: fetchParseStoreSummary, keyDecisions: fetchParseStoreDecisions},
		{traceName: "seed-noise-and-strategies/fetch-parse-store-2", summary: fetchParseStoreSummary, keyDecisions: fetchParseStoreDecisions},
		{traceName: "seed-noise-and-strategies/fetch-parse-store-3", summary: fetchParseStoreSummary, keyDecisions: fetchParseStoreDecisions},
		{traceName: "seed-noise-and-strategies/fetch-store-1", summary: fetchStoreSummary, keyDecisions: fetchStoreDecisions},
		{traceName: "seed-noise-and-strategies/fetch-store-2", summary: fetchStoreSummary, keyDecisions: fetchStoreDecisions},
		{traceName: "seed-noise-and-strategies/outlier-bash-fetch", summary: outlierBashFetchSummary, keyDecisions: outlierBashFetchDecisions},
	}

	out := make([]transcriptEntry, 0, len(specs))
	for _, s := range specs {
		traceSteps, ok := steps[s.traceName]
		if !ok {
			t.Fatalf("transcript spec: trace %q not in seeded DB", s.traceName)
		}
		out = append(out, transcriptEntry{
			TraceName:    s.traceName,
			PromptsHash:  handlers.TranscriptPromptsHash(traceSteps),
			Summary:      s.summary,
			KeyDecisions: s.keyDecisions,
		})
	}
	return out
}

// buildEmbeddingEntries calls Voyage AI for each seeded trace and returns the
// vectors. Skipped (returns nil) when VOYAGE_API_KEY is not set, so the
// triage/transcript half of the generator can run without external dependencies.
func buildEmbeddingEntries(t *testing.T, steps map[string][]snapshot.Step) []embeddingEntry {
	t.Helper()

	apiKey := os.Getenv("VOYAGE_API_KEY")
	if apiKey == "" {
		t.Log("VOYAGE_API_KEY not set; skipping embedding generation")
		return nil
	}

	model := os.Getenv("VOYAGE_MODEL")
	if model == "" {
		model = "voyage-3-lite"
	}

	// Sort names for deterministic output ordering in seed-cache.json.
	names := make([]string, 0, len(steps))
	for name := range steps {
		names = append(names, name)
	}
	sort.Strings(names)

	// Voyage free tier without a payment method caps at 3 RPM. Sleep 21s
	// between calls to stay under the limit; the free-token allowance (200M
	// tokens for Voyage 3 series) is wildly more than this generator needs.
	// Dedupe by rendered text first: identical step content (e.g., the 5
	// identical runs in seed-tool-order-stable) produces identical text and
	// thus identical Voyage output, so one API call covers all matching traces.
	const throttle = 21 * time.Second

	textByName := make(map[string]string, len(names))
	uniqueTexts := make([]string, 0, len(names))
	seen := make(map[string]bool)
	for _, name := range names {
		text := handlers.StepsToText(steps[name])
		textByName[name] = text
		if !seen[text] {
			seen[text] = true
			uniqueTexts = append(uniqueTexts, text)
		}
	}
	t.Logf("voyage: %d traces collapse to %d unique texts", len(names), len(uniqueTexts))

	vecByText := make(map[string][]float32, len(uniqueTexts))
	for i, text := range uniqueTexts {
		if i > 0 {
			time.Sleep(throttle)
		}
		vec, err := voyageEmbed(apiKey, model, text)
		if err != nil {
			t.Fatalf("voyage embed unique text %d/%d: %v", i+1, len(uniqueTexts), err)
		}
		vecByText[text] = vec
		t.Logf("embedded unique %d/%d (dim=%d)", i+1, len(uniqueTexts), len(vec))
	}

	out := make([]embeddingEntry, 0, len(names))
	for _, name := range names {
		out = append(out, embeddingEntry{
			TraceName: name,
			Vector:    vecByText[textByName[name]],
			ModelName: model,
		})
	}
	t.Logf("populated %d embedding entries from %d Voyage calls", len(out), len(uniqueTexts))
	return out
}

// voyageEmbed posts a single trace's rendered text to the Voyage API and
// returns the vector. Mirrors handlers.VoyageEmbedder.Embed but stripped
// down for the generator's one-text-at-a-time use.
func voyageEmbed(apiKey, model, text string) ([]float32, error) {
	body, err := json.Marshal(map[string]interface{}{
		"input": []string{text},
		"model": model,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost,
		"https://api.voyageai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voyage call: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("voyage %d: %s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("voyage returned no embedding")
	}
	return parsed.Data[0].Embedding, nil
}
