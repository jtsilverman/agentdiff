package diff

// Verdict represents the outcome of a diff comparison.
type Verdict string

const (
	VerdictPass       Verdict = "pass"
	VerdictChanged    Verdict = "changed"
	VerdictRegression Verdict = "regression"
)

// DiffResult holds the complete comparison between two snapshots.
type DiffResult struct {
	Snapshot1 string          `json:"snapshot_1"`
	Snapshot2 string          `json:"snapshot_2"`
	Overall   Verdict         `json:"overall"`
	ToolDiff  ToolDiffResult  `json:"tool_diff"`
	TextDiff  TextDiffResult  `json:"text_diff"`
	StepsDiff StepsDiffResult `json:"steps_diff"`
}

// ToolDiffResult captures differences in tool usage between two traces.
type ToolDiffResult struct {
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Reordered bool     `json:"reordered"`
	EditDist  int      `json:"edit_distance"`
	Score     float64  `json:"score"` // 0.0 = identical, 1.0 = completely different
}

// TextDiffResult captures differences in text content between two traces.
type TextDiffResult struct {
	Similarity float64 `json:"similarity"` // 0.0 = unrelated, 1.0 = identical
	Score      float64 `json:"score"`      // inverted: 0.0 = identical, 1.0 = completely different
}

// StepsDiffResult captures differences in step counts between two traces.
type StepsDiffResult struct {
	CountA int `json:"count_a"`
	CountB int `json:"count_b"`
	Delta  int `json:"delta"`
}
