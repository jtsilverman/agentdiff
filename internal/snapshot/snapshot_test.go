package snapshot

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSnapshotJSONRoundTrip(t *testing.T) {
	snap := Snapshot{
		ID:        "abc123def456",
		Name:      "test-snap",
		Source:    "claude",
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Metadata:  map[string]string{"model": "claude-4", "version": "1.0"},
		Steps: []Step{
			{Role: "user", Content: "hello"},
			{
				Role:    "assistant",
				Content: "I'll read that file.",
				ToolCall: &ToolCall{
					Name: "read_file",
					Args: map[string]interface{}{"path": "/tmp/test.txt"},
				},
			},
			{
				Role: "tool",
				ToolResult: &ToolResult{
					Name:   "read_file",
					Output: "file contents here",
				},
			},
		},
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Snapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != snap.ID {
		t.Errorf("ID = %q, want %q", got.ID, snap.ID)
	}
	if got.Name != snap.Name {
		t.Errorf("Name = %q, want %q", got.Name, snap.Name)
	}
	if got.Source != snap.Source {
		t.Errorf("Source = %q, want %q", got.Source, snap.Source)
	}
	if !got.Timestamp.Equal(snap.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, snap.Timestamp)
	}
	if len(got.Steps) != len(snap.Steps) {
		t.Fatalf("Steps len = %d, want %d", len(got.Steps), len(snap.Steps))
	}
	if got.Steps[1].ToolCall == nil {
		t.Fatal("Steps[1].ToolCall is nil")
	}
	if got.Steps[1].ToolCall.Name != "read_file" {
		t.Errorf("ToolCall.Name = %q, want %q", got.Steps[1].ToolCall.Name, "read_file")
	}
	if got.Steps[2].ToolResult == nil {
		t.Fatal("Steps[2].ToolResult is nil")
	}
	if got.Steps[2].ToolResult.Output != "file contents here" {
		t.Errorf("ToolResult.Output = %q, want %q", got.Steps[2].ToolResult.Output, "file contents here")
	}
	if got.Steps[2].ToolResult.IsError {
		t.Error("ToolResult.IsError should be false")
	}
}

func TestToolCallOmitEmpty(t *testing.T) {
	step := Step{Role: "user", Content: "hello"}
	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if contains(s, "tool_call") {
		t.Error("tool_call should be omitted when nil")
	}
	if contains(s, "tool_result") {
		t.Error("tool_result should be omitted when nil")
	}
}

func TestToolResultIsError(t *testing.T) {
	tr := ToolResult{Name: "bash", Output: "exit code 1", IsError: true}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.IsError {
		t.Error("IsError should be true")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
