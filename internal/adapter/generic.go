package adapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// GenericAdapter parses arbitrary JSONL traces using configurable field mappings.
type GenericAdapter struct {
	cfg config.GenericAdapterConfig
}

// NewGenericAdapter creates a GenericAdapter with the given field mappings.
func NewGenericAdapter(cfg config.GenericAdapterConfig) *GenericAdapter {
	return &GenericAdapter{cfg: cfg}
}

// Parse reads JSONL input and returns normalized Steps and metadata.
func (g *GenericAdapter) Parse(input []byte) ([]snapshot.Step, map[string]string, error) {
	var steps []snapshot.Step
	meta := map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(input))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(line, &obj); err != nil {
			return nil, nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}

		step, ok := g.parseLine(obj)
		if ok {
			steps = append(steps, step)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading input: %w", err)
	}

	return steps, meta, nil
}

// parseLine converts a single JSON object into a Step using the configured field mappings.
func (g *GenericAdapter) parseLine(obj map[string]interface{}) (snapshot.Step, bool) {
	// Resolve the role field.
	rawRole, ok := resolveFieldPath(obj, g.cfg.RoleField)
	if !ok {
		return snapshot.Step{}, false
	}
	roleStr, ok := rawRole.(string)
	if !ok {
		return snapshot.Step{}, false
	}

	// Map through role_map if configured.
	role := roleStr
	if g.cfg.RoleMap != nil {
		if mapped, ok := g.cfg.RoleMap[roleStr]; ok {
			role = mapped
		}
	}

	// Validate role is one of the known types.
	switch role {
	case "user", "assistant", "tool_call", "tool_result":
		// valid
	default:
		return snapshot.Step{}, false
	}

	switch role {
	case "tool_call":
		name := resolveString(obj, g.cfg.ToolNameField)
		args := resolveMap(obj, g.cfg.ToolArgsField)
		return snapshot.Step{
			Role: "tool_call",
			ToolCall: &snapshot.ToolCall{
				Name: name,
				Args: args,
			},
		}, true

	case "tool_result":
		name := resolveString(obj, g.cfg.ToolNameField)
		output := resolveString(obj, g.cfg.ToolOutputField)
		return snapshot.Step{
			Role: "tool_result",
			ToolResult: &snapshot.ToolResult{
				Name:   name,
				Output: output,
			},
		}, true

	default: // user, assistant
		content := resolveString(obj, g.cfg.ContentField)
		return snapshot.Step{
			Role:    role,
			Content: content,
		}, true
	}
}

// resolveFieldPath walks a nested map using a dot-notation path and returns the value.
func resolveFieldPath(obj map[string]interface{}, path string) (interface{}, bool) {
	if path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current interface{} = obj
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// resolveString resolves a field path and returns its string value, or empty string.
func resolveString(obj map[string]interface{}, path string) string {
	v, ok := resolveFieldPath(obj, path)
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// resolveMap resolves a field path and returns its map value, or nil.
func resolveMap(obj map[string]interface{}, path string) map[string]interface{} {
	v, ok := resolveFieldPath(obj, path)
	if !ok {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}
