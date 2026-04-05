package report

import (
	"encoding/json"
	"io"

	"github.com/jtsilverman/agentdiff/internal/diff"
)

// JSON writes the diff result as indented JSON to w.
func JSON(result diff.DiffResult, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
