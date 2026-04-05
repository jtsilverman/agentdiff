package adapter

import (
	"encoding/json"
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// AgentsSdkAdapter parses OpenAI Agents SDK trace JSON into Steps.
type AgentsSdkAdapter struct{}

// agentsSdkTrace represents the top-level trace format.
type agentsSdkTrace struct {
	TraceID string            `json:"trace_id"`
	Spans   []json.RawMessage `json:"spans"`
}

// agentsSdkSpan represents a single span in the trace tree.
type agentsSdkSpan struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	SpanData json.RawMessage   `json:"span_data"`
	Children []json.RawMessage `json:"children"`
}

const maxRecursionDepth = 100

// Parse reads an OpenAI Agents SDK trace and returns normalized Steps and metadata.
func (a *AgentsSdkAdapter) Parse(input []byte) ([]snapshot.Step, map[string]string, error) {
	var trace agentsSdkTrace
	if err := json.Unmarshal(input, &trace); err != nil {
		return nil, nil, fmt.Errorf("agents_sdk: invalid JSON: %w", err)
	}

	meta := map[string]string{}
	if trace.TraceID != "" {
		meta["trace_id"] = trace.TraceID
	}

	var steps []snapshot.Step
	for _, raw := range trace.Spans {
		s, err := a.walkSpan(raw, meta, 0)
		if err != nil {
			return nil, nil, err
		}
		steps = append(steps, s...)
	}

	return steps, meta, nil
}

// walkSpan recursively processes a span and its children.
func (a *AgentsSdkAdapter) walkSpan(raw json.RawMessage, meta map[string]string, depth int) ([]snapshot.Step, error) {
	if depth >= maxRecursionDepth {
		return nil, fmt.Errorf("agents_sdk: recursion depth exceeded %d levels", maxRecursionDepth)
	}

	if raw == nil {
		return nil, nil
	}

	var span agentsSdkSpan
	if err := json.Unmarshal(raw, &span); err != nil {
		return nil, nil
	}

	// Parse span_data into a generic map. Skip span if missing or not an object.
	if span.SpanData == nil {
		return nil, nil
	}
	var spanData map[string]interface{}
	if err := json.Unmarshal(span.SpanData, &spanData); err != nil {
		return nil, nil
	}

	var steps []snapshot.Step

	switch span.Type {
	case "function":
		steps = append(steps, a.parseFunctionSpan(span.Name, spanData)...)

	case "agent":
		if model, ok := spanData["model"].(string); ok && model != "" {
			meta["model"] = model
		}
		for _, child := range span.Children {
			if child == nil {
				continue
			}
			childSteps, err := a.walkSpan(child, meta, depth+1)
			if err != nil {
				return nil, err
			}
			steps = append(steps, childSteps...)
		}

	case "generation":
		if output, ok := spanData["output"].(string); ok && output != "" {
			steps = append(steps, snapshot.Step{
				Role:    "assistant",
				Content: output,
			})
		}

	default:
		// Unknown type: skip silently.
	}

	return steps, nil
}

// parseFunctionSpan emits a tool_call step followed by a tool_result step.
func (a *AgentsSdkAdapter) parseFunctionSpan(name string, spanData map[string]interface{}) []snapshot.Step {
	// Build args from span_data.input.
	args := map[string]interface{}{}
	if input, ok := spanData["input"]; ok {
		switch v := input.(type) {
		case map[string]interface{}:
			args = v
		case string:
			args = map[string]interface{}{"input": v}
		}
	}

	// Build output from span_data.output.
	output := ""
	if out, ok := spanData["output"]; ok {
		switch v := out.(type) {
		case string:
			output = v
		default:
			b, _ := json.Marshal(v)
			output = string(b)
		}
	}

	return []snapshot.Step{
		{
			Role: "tool_call",
			ToolCall: &snapshot.ToolCall{
				Name: name,
				Args: args,
			},
		},
		{
			Role: "tool_result",
			ToolResult: &snapshot.ToolResult{
				Name:   name,
				Output: output,
			},
		},
	}
}
