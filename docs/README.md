# docs/

The four short docs a session reads to learn AgentDiff's current shape.

| File | Question it answers |
|---|---|
| `architecture/three-surfaces.md` | The CLI, the web API, and the Action, what each owns, and the packages all three share |
| `conventions/go-and-tests.md` | Where a test goes, how a handler takes its dependencies, how a commit is named |
| `runbooks/build-test-deploy.md` | Run the tests, run the dashboard locally, run the bench suite, deploy the API |
| `decisions/two-dimension-diff.md` | Why a diff is Levenshtein on tools plus Jaccard on text, and not an assertion |
| `decisions/llm-behind-an-interface.md` | Why every LLM endpoint sits behind an interface with a deterministic fallback |
| `decisions/static-site-over-nextjs.md` | Why the Next.js frontend was deleted and the Go binary serves plain HTML |

`README.md` at the repo root is the product pitch and the feature list. It is ahead of the code in
one place and behind it in two; `architecture/three-surfaces.md` names both. `SPEC.md`,
`SPEC-web.md`, and `SPEC-bench.md` are the historical specs for the CLI, the dashboard, and the
bench suite.
