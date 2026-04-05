package adapter

import (
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// Adapter parses raw agent trace data into normalized Steps and metadata.
type Adapter interface {
	Parse(input []byte) ([]snapshot.Step, map[string]string, error)
}

// registry holds all registered adapters by name.
var registry = map[string]Adapter{}

func init() {
	registry["claude"] = &ClaudeAdapter{}
	registry["openai"] = &OpenAIAdapter{}
}

// Get returns the adapter registered under the given name.
func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown adapter: %q", name)
	}
	return a, nil
}
