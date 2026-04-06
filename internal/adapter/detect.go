package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Detect examines the input bytes and returns the appropriate adapter.
// It peeks at the JSON structure without fully parsing the data.
func Detect(input []byte) (Adapter, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	switch trimmed[0] {
	case '[':
		// JSON array: check if elements have "role" fields (OpenAI format).
		if looksLikeOpenAIArray(trimmed) {
			return &OpenAIAdapter{}, nil
		}
		return nil, fmt.Errorf("unrecognized JSON array format")

	case '{':
		// JSON object: check for known top-level keys.
		// If it fails as a single object, fall through to JSONL check.
		a, err := detectJSONObject(trimmed)
		if err == nil {
			return a, nil
		}
		// Might be JSONL (multiple lines starting with '{').
		if looksLikeJSONL(trimmed) {
			return detectJSONLFormat(trimmed), nil
		}
		return nil, err

	default:
		// Check for JSONL (newline-separated JSON objects).
		if looksLikeJSONL(trimmed) {
			return detectJSONLFormat(trimmed), nil
		}
		return nil, fmt.Errorf("unrecognized format: not JSON array, object, or JSONL")
	}
}

// looksLikeOpenAIArray checks if a JSON array contains objects with "role" fields.
func looksLikeOpenAIArray(data []byte) bool {
	// Parse just enough to check the first element.
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil || len(raw) == 0 {
		return false
	}
	var peek struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(raw[0], &peek); err != nil {
		return false
	}
	return peek.Role != ""
}

// detectJSONObject routes a JSON object to the correct adapter based on keys.
func detectJSONObject(data []byte) (Adapter, error) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}

	// Claude Code stream-json: has "message" key or is a "system" init line with "subtype".
	if _, hasMessage := keys["message"]; hasMessage {
		return &ClaudeCodeAdapter{}, nil
	}
	if raw, hasType := keys["type"]; hasType {
		var typeStr string
		if json.Unmarshal(raw, &typeStr) == nil && typeStr == "system" {
			if _, hasSubtype := keys["subtype"]; hasSubtype {
				return &ClaudeCodeAdapter{}, nil
			}
		}
	}

	// Agents SDK: has trace_id and spans keys.
	if _, hasTraceID := keys["trace_id"]; hasTraceID {
		if _, hasSpans := keys["spans"]; hasSpans {
			return &AgentsSdkAdapter{}, nil
		}
	}

	// "choices" key -> OpenAI API response.
	if _, ok := keys["choices"]; ok {
		return &OpenAIAdapter{}, nil
	}

	// "messages" key -> OpenAI messages wrapper.
	if _, ok := keys["messages"]; ok {
		return &OpenAIAdapter{}, nil
	}

	// Otherwise try Claude (single JSONL object).
	return &ClaudeAdapter{}, nil
}

// detectJSONLFormat peeks at the first JSONL line to distinguish LangChain from Claude.
func detectJSONLFormat(data []byte) Adapter {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(line, &obj); err != nil {
			break
		}
		// Claude Code stream-json: first line has "message" key or is system init with "subtype".
		if _, hasMessage := obj["message"]; hasMessage {
			return &ClaudeCodeAdapter{}
		}
		if raw, hasType := obj["type"]; hasType {
			var typeStr string
			if json.Unmarshal(raw, &typeStr) == nil && typeStr == "system" {
				if _, hasSubtype := obj["subtype"]; hasSubtype {
					return &ClaudeCodeAdapter{}
				}
			}
		}
		// LangChain JSONL: first line has run_id and type starting with "on_".
		if _, hasRunID := obj["run_id"]; hasRunID {
			if raw, hasType := obj["type"]; hasType {
				var typeStr string
				if err := json.Unmarshal(raw, &typeStr); err == nil {
					if strings.HasPrefix(typeStr, "on_") {
						return &LangChainAdapter{}
					}
				}
			}
		}
		break
	}
	return &ClaudeAdapter{}
}

// looksLikeJSONL checks if the data contains newline-separated JSON objects.
func looksLikeJSONL(data []byte) bool {
	lines := bytes.Split(data, []byte("\n"))
	validCount := 0
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] != '{' {
			return false
		}
		if !json.Valid(line) {
			return false
		}
		validCount++
	}
	return validCount >= 1
}
