package adapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// LangChainAdapter parses LangChain callback JSONL into Steps.
type LangChainAdapter struct{}

// langchainEvent represents a single callback event line.
type langchainEvent struct {
	Type        string                 `json:"type"`
	RunID       string                 `json:"run_id"`
	ParentRunID string                 `json:"parent_run_id,omitempty"`
	Name        string                 `json:"name"`
	Inputs      json.RawMessage        `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
}

// Parse reads LangChain callback JSONL and returns normalized Steps and metadata.
func (l *LangChainAdapter) Parse(input []byte) ([]snapshot.Step, map[string]string, error) {
	meta := map[string]string{}

	if len(bytes.TrimSpace(input)) == 0 {
		return nil, meta, nil
	}

	var steps []snapshot.Step
	seenChainStart := false

	scanner := bufio.NewScanner(bytes.NewReader(input))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var ev langchainEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}

		switch ev.Type {
		case "on_chain_start":
			if !seenChainStart {
				seenChainStart = true
				if ev.Name != "" {
					meta["agent_name"] = ev.Name
				}
			}
			// Skip as step.

		case "on_tool_start":
			args, err := parseInputsAsMap(ev.Inputs)
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: tool_start inputs: %w", lineNum, err)
			}
			steps = append(steps, snapshot.Step{
				Role: "tool_call",
				ToolCall: &snapshot.ToolCall{
					Name: ev.Name,
					Args: args,
				},
			})

		case "on_tool_end":
			output := extractToolOutput(ev.Outputs)
			steps = append(steps, snapshot.Step{
				Role: "tool_result",
				ToolResult: &snapshot.ToolResult{
					Name:   ev.Name,
					Output: output,
				},
			})

		case "on_llm_end":
			content := extractLLMContent(ev.Outputs)
			if content == "" {
				continue
			}
			steps = append(steps, snapshot.Step{
				Role:    "assistant",
				Content: content,
			})

		case "on_chain_end", "on_llm_start":
			// Skip.

		default:
			// Unknown type: skip.
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading input: %w", err)
	}

	return steps, meta, nil
}

// parseInputsAsMap parses the inputs field. If it's a JSON object, returns it
// as a map. Otherwise wraps the value as {"input": value}.
func parseInputsAsMap(raw json.RawMessage) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}

	// Try as map first.
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err == nil {
		return m, nil
	}

	// Not a map: wrap as {"input": value}.
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return map[string]interface{}{"input": v}, nil
}

// extractToolOutput extracts the output string from a tool_end outputs map.
// Uses outputs.output if present, otherwise JSON-stringifies the entire map.
func extractToolOutput(outputs map[string]interface{}) string {
	if outputs == nil {
		return ""
	}
	if out, ok := outputs["output"]; ok {
		if s, ok := out.(string); ok {
			return s
		}
	}
	// Stringify entire outputs map.
	b, _ := json.Marshal(outputs)
	return string(b)
}

// extractLLMContent extracts text from an llm_end outputs map.
// Tries outputs.generations[0][0].text first, then outputs.output.
func extractLLMContent(outputs map[string]interface{}) string {
	if outputs == nil {
		return ""
	}

	// Try generations[0][0].text
	if gens, ok := outputs["generations"]; ok {
		if outer, ok := gens.([]interface{}); ok && len(outer) > 0 {
			if inner, ok := outer[0].([]interface{}); ok && len(inner) > 0 {
				if gen, ok := inner[0].(map[string]interface{}); ok {
					if text, ok := gen["text"].(string); ok {
						return text
					}
				}
			}
		}
	}

	// Try outputs.output
	if out, ok := outputs["output"]; ok {
		if s, ok := out.(string); ok {
			return s
		}
	}

	return ""
}
