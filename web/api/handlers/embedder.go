package handlers

import (
	"context"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// EmbedRequest is the input to an Embedder.Embed call.
// TraceName + Steps give the production embedder the text content to vectorize.
// The fakeEmbedder in tests keys off TraceName for deterministic geometry.
type EmbedRequest struct {
	TraceID   string
	TraceName string
	Steps     []snapshot.Step
}

// EmbedResult is the output of an Embedder.Embed call.
type EmbedResult struct {
	Vector    []float32
	ModelName string
}

// Embedder turns a trace's step sequence into a vector. The interface is the
// DI seam between the trace-insert side-effect and the production Voyage call
// (or a fake in tests). Nil-receiver semantics: a nil Embedder means embedding
// generation is off — POST /traces still works, no row written to
// trace_embeddings, and /similar naturally returns empty matches for that trace.
type Embedder interface {
	Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error)
}
