package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
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
			return &ClaudeAdapter{}, nil
		}
		return nil, err

	default:
		// Check for JSONL (newline-separated JSON objects) -> Claude format.
		if looksLikeJSONL(trimmed) {
			return &ClaudeAdapter{}, nil
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
