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
// LLM content for the 3 seeded task-driven baselines (api-endpoint-rename /
// auth-migration-md5-to-bcrypt / new-endpoint-with-tests). Triage + transcript
// portions only; embeddings are appended in a separate Voyage curl pass.
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
		// seed-api-endpoint-rename: three coexisting strategies for the same task
		// (grep-first / search-first / assume-known-path) — every comparison is
		// healthy variance, no regression.
		{
			a:              "seed-api-endpoint-rename/run-1",
			b:              "seed-api-endpoint-rename/run-3",
			summary:        "Run-1 used grep to enumerate /users call sites before editing; run-3 used the higher-level search tool to find endpoint definitions. Both arrived at the same set of files but via different discovery primitives.",
			classification: "variance",
			likelyCause:    "Both runs are valid completions. The choice between grep (literal pattern match) and search (semantic file query) likely depends on how the agent interpreted the task — whether 'rename across the codebase' implied a literal string sweep or a structured-route enumeration. Healthy strategy variance, not a defect.",
		},
		{
			a:              "seed-api-endpoint-rename/run-1",
			b:              "seed-api-endpoint-rename/run-5",
			summary:        "Run-1 used grep before editing; run-5 skipped enumeration entirely and went straight to read_file on the route file. Run-5 assumed the route definition's location instead of confirming it.",
			classification: "variance",
			likelyCause:    "Run-5's assume-known-path strategy is risky when the codebase doesn't match the agent's assumption, but in this case both runs ended up editing the same files. The strategy variance is the demo signal — three different exploration depths produce equivalent outcomes here, but in a less stable codebase the assume-known path would silently miss call sites.",
		},
		{
			a:              "seed-api-endpoint-rename/run-3",
			b:              "seed-api-endpoint-rename/run-5",
			summary:        "Run-3 used search to enumerate endpoint definitions across the codebase; run-5 used no discovery tool at all and went directly to the known route file.",
			classification: "variance",
			likelyCause:    "Cross-strategy comparison between the most-cautious (search-first) and least-cautious (assume-known) approaches in this baseline. Both terminate at write_file; the divergence is in the discovery step, not in the edit step itself.",
		},
		{
			a:              "seed-api-endpoint-rename/run-1",
			b:              "seed-api-endpoint-rename/run-2",
			summary:        "Both runs follow the identical grep → read_file → write_file → write_file sequence. Within-strategy reproducibility check.",
			classification: "variance",
			likelyCause:    "Within-strategy stability check. The agent behaves consistently across multiple invocations when it picks the grep-first strategy, so any divergence observed against runs 3/4/5 is attributable to strategy choice rather than nondeterminism within one strategy.",
		},

		// seed-auth-migration-md5-to-bcrypt: before-prompt path uses search to
		// enumerate call sites; after-prompt path skips search and silently
		// misses sites. Classic regression signal.
		{
			a:              "seed-auth-migration-md5-to-bcrypt/before-1",
			b:              "seed-auth-migration-md5-to-bcrypt/after-prompt-change",
			summary:        "Run before-1 reads the password module, searches for every MD5 call site, reads each, and rewrites them. The after-prompt-change run reads only the password module and rewrites it, omitting the search step that previously enumerated other call sites.",
			classification: "regression",
			likelyCause:    "The prompt change appears to have removed the directive that signaled comprehensive call-site updates were needed. Because the search step was load-bearing (it surfaced src/auth/verify.go and src/migration/users.go), dropping it leaves MD5 references in place at those call sites — a silent quality drop that compiles fine but is functionally incorrect. Recommend reinstating the discovery-then-edit pattern in the prompt.",
		},
		{
			a:              "seed-auth-migration-md5-to-bcrypt/before-1",
			b:              "seed-auth-migration-md5-to-bcrypt/before-2",
			summary:        "Both runs follow the identical read → search → read → write → write sequence. The agent reproduces the discovery-first migration consistently across pre-change invocations.",
			classification: "variance",
			likelyCause:    "Within-baseline reproducibility check. The pre-change prompt elicits the same comprehensive migration shape every time, so the divergence observed in after-prompt-change is fully attributable to the prompt edit rather than to nondeterminism.",
		},

		// seed-new-endpoint-with-tests: 3 baseline runs add the route directly;
		// 2 later runs adopt a test-first pattern that ALSO writes a test file.
		// Additive — original capability preserved, new capability layered on top.
		{
			a:              "seed-new-endpoint-with-tests/run-1",
			b:              "seed-new-endpoint-with-tests/run-4",
			summary:        "Run-1 added the /preferences route in a single write to the routes file. Run-4 added the route AND wrote a companion test file (src/routes/users_test.go) that did not exist in run-1.",
			classification: "additive",
			likelyCause:    "The agent has adopted a test-first or test-alongside pattern that was absent from the baseline runs. This is additive behavior rather than a regression: the original route-addition capability is preserved, and a new test-writing capability has been layered on top. Worth verifying that the test file exercises the new endpoint and isn't just a stub.",
		},
		{
			a:              "seed-new-endpoint-with-tests/run-4",
			b:              "seed-new-endpoint-with-tests/run-5",
			summary:        "Both runs use the new route-plus-test pattern consistently: read the routes file, add the new route, then create the companion test file. The test-alongside pattern is stable across multiple invocations once the agent adopted it.",
			classification: "variance",
			likelyCause:    "Post-adoption stability check. The agent has settled on the test-alongside workflow, suggesting either a stable prompt change or a learned-preference for the new pattern. No further divergence within the post-adoption cohort.",
		},
		{
			a:              "seed-new-endpoint-with-tests/run-1",
			b:              "seed-new-endpoint-with-tests/run-2",
			summary:        "Both runs follow the identical read_file → write_file sequence to add the new route. No divergence between baseline runs.",
			classification: "variance",
			likelyCause:    "Within-baseline reproducibility check on the pre-adoption cohort. The agent behaves consistently when it picks the route-only pattern, so any divergence observed against runs 4/5 is attributable to the additive pattern shift rather than to nondeterminism.",
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

	// Per-scenario templates by content shape. The 3 task-driven scenarios
	// produce 5 distinct shapes total; reusing template text per shape keeps
	// the demo readable without claiming per-run insight the model wouldn't
	// actually have. Shape captions name the scenario context so identical
	// tool sequences in different scenarios (read→write) get scenario-honest
	// prose, not generic templates.

	// seed-api-endpoint-rename shapes.
	apiGrepFirstSummary := "The agent used grep to enumerate every /users call site in the codebase before touching any file, then opened the route file and edited it, then edited the client file. This is the cautious 'enumerate-then-edit' pattern: it pays one extra discovery call up front to avoid missing call sites later."
	apiGrepFirstDecisions := []string{
		"Used grep before any read or write so the edit plan was anchored on the full set of call sites, not just the first one the agent recalled.",
		"Edited the route file and the client file in separate write_file calls rather than combining them, keeping each edit's diff small and reviewable.",
	}

	apiSearchFirstSummary := "The agent invoked the higher-level search tool to locate endpoint definitions across the codebase, then opened the route file and rewrote it. The search step is semantic (search for endpoint-shaped patterns) rather than literal (grep for the string), suggesting the agent reasoned about the task at a structural level."
	apiSearchFirstDecisions := []string{
		"Chose search over grep, treating the task as finding route definitions rather than finding a literal string.",
		"Stopped after editing the primary route file, trusting search's enumeration to be exhaustive without a separate verification pass.",
	}

	apiAssumeKnownSummary := "The agent skipped all discovery and went straight to read_file on the route file, then write_file on the route file, then write_file on the client file. This 'assume-known-path' pattern is the fastest of the three strategies; it's also the riskiest in a codebase where the assumption doesn't hold."
	apiAssumeKnownDecisions := []string{
		"Skipped both grep and search, betting that the route file's location was canonical and that the client file would also need updating.",
		"Edited route and client in two separate writes, suggesting the agent had a pre-formed model of the call-site fanout without explicit discovery.",
	}

	// seed-auth-migration-md5-to-bcrypt shapes.
	authBeforeSummary := "The agent read the password module to understand its current shape, then searched the entire src/ tree for MD5 call sites (surfacing src/auth/verify.go and src/migration/users.go), then read verify.go to confirm its hash-comparison pattern, then rewrote both passwords.go and verify.go to use bcrypt. The discovery step is load-bearing: without it, the agent would have updated only the module it was pointed at."
	authBeforeDecisions := []string{
		"Inserted a search step between the initial read and the edits, treating the task as requiring comprehensive call-site updates rather than a single-file rewrite.",
		"Read every call site before writing, ensuring the bcrypt API contract (GenerateFromPassword for hashing, CompareHashAndPassword for verification) matched each site's usage.",
		"Issued two separate write_file calls — one per affected module — rather than a combined edit, keeping each diff focused.",
	}

	authAfterSummary := "The agent read passwords.go and rewrote it to use bcrypt, then stopped. No search step, no discovery of other call sites. The prompt change that produced this run appears to have de-emphasized the comprehensive-update requirement; the result is a silent regression where verify.go and migration/users.go still call md5.Sum after the migration."
	authAfterDecisions := []string{
		"Skipped the search step that earlier runs used to enumerate MD5 call sites, leaving sites outside passwords.go untouched.",
		"Edited only the file the prompt directly named, treating the task as a single-file rewrite rather than a codebase-wide migration.",
	}

	// seed-new-endpoint-with-tests shapes.
	newEndpointRouteOnlySummary := "The agent read the routes file to understand the existing pattern, then added the new /users/:id/preferences route in a single write. No test file was created; the task was treated as a pure routing change."
	newEndpointRouteOnlyDecisions := []string{
		"Read the routes file before editing to match the existing route-declaration style.",
		"Added the new route in a single write_file call, leaving testing as a presumed-separate concern.",
	}

	newEndpointWithTestSummary := "The agent read the routes file to understand the existing pattern, added the new /users/:id/preferences route, then created a companion test file (src/routes/users_test.go) for the new endpoint. The test-alongside step is additive: the route is added either way, and this run also produces a test stub."
	newEndpointWithTestDecisions := []string{
		"Read the routes file before editing to match the existing route-declaration style.",
		"Added the new route in a single write to the routes file.",
		"Created a companion test file (users_test.go) after adding the route, layering a test-first habit on top of the basic route-addition workflow.",
	}

	specs := []transcriptSpec{
		// seed-api-endpoint-rename — runs 1-2 grep-first, runs 3-4 search-first, run-5 assume-known.
		{traceName: "seed-api-endpoint-rename/run-1", summary: apiGrepFirstSummary, keyDecisions: apiGrepFirstDecisions},
		{traceName: "seed-api-endpoint-rename/run-2", summary: apiGrepFirstSummary, keyDecisions: apiGrepFirstDecisions},
		{traceName: "seed-api-endpoint-rename/run-3", summary: apiSearchFirstSummary, keyDecisions: apiSearchFirstDecisions},
		{traceName: "seed-api-endpoint-rename/run-4", summary: apiSearchFirstSummary, keyDecisions: apiSearchFirstDecisions},
		{traceName: "seed-api-endpoint-rename/run-5", summary: apiAssumeKnownSummary, keyDecisions: apiAssumeKnownDecisions},

		// seed-auth-migration-md5-to-bcrypt — before-1..4 discovery-first, after-prompt-change skips search.
		{traceName: "seed-auth-migration-md5-to-bcrypt/before-1", summary: authBeforeSummary, keyDecisions: authBeforeDecisions},
		{traceName: "seed-auth-migration-md5-to-bcrypt/before-2", summary: authBeforeSummary, keyDecisions: authBeforeDecisions},
		{traceName: "seed-auth-migration-md5-to-bcrypt/before-3", summary: authBeforeSummary, keyDecisions: authBeforeDecisions},
		{traceName: "seed-auth-migration-md5-to-bcrypt/before-4", summary: authBeforeSummary, keyDecisions: authBeforeDecisions},
		{traceName: "seed-auth-migration-md5-to-bcrypt/after-prompt-change", summary: authAfterSummary, keyDecisions: authAfterDecisions},

		// seed-new-endpoint-with-tests — runs 1-3 route-only, runs 4-5 route + test.
		{traceName: "seed-new-endpoint-with-tests/run-1", summary: newEndpointRouteOnlySummary, keyDecisions: newEndpointRouteOnlyDecisions},
		{traceName: "seed-new-endpoint-with-tests/run-2", summary: newEndpointRouteOnlySummary, keyDecisions: newEndpointRouteOnlyDecisions},
		{traceName: "seed-new-endpoint-with-tests/run-3", summary: newEndpointRouteOnlySummary, keyDecisions: newEndpointRouteOnlyDecisions},
		{traceName: "seed-new-endpoint-with-tests/run-4", summary: newEndpointWithTestSummary, keyDecisions: newEndpointWithTestDecisions},
		{traceName: "seed-new-endpoint-with-tests/run-5", summary: newEndpointWithTestSummary, keyDecisions: newEndpointWithTestDecisions},
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
