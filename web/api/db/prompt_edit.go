package db

import (
	"fmt"

	"github.com/google/uuid"
)

// PromptEditRun is one persisted prompt-edit replay linking an original trace
// to a re-run trace that diverged at step_index after modified_prompt was substituted.
type PromptEditRun struct {
	ID              string
	OriginalTraceID string
	EditedTraceID   string
	StepIndex       int
	ModifiedPrompt  string
}

// CreatePromptEditRun inserts a prompt_edit_runs row and returns it.
// Returns an error wrapping FK violations when either trace ID is unknown.
func (db *DB) CreatePromptEditRun(originalTraceID, editedTraceID string, stepIndex int, modifiedPrompt string) (PromptEditRun, error) {
	id := uuid.New().String()
	_, err := db.conn.Exec(
		`INSERT INTO prompt_edit_runs (id, original_trace_id, edited_trace_id, step_index, modified_prompt)
		 VALUES (?, ?, ?, ?, ?)`,
		id, originalTraceID, editedTraceID, stepIndex, modifiedPrompt,
	)
	if err != nil {
		return PromptEditRun{}, fmt.Errorf("insert prompt_edit_run: %w", err)
	}
	return PromptEditRun{
		ID:              id,
		OriginalTraceID: originalTraceID,
		EditedTraceID:   editedTraceID,
		StepIndex:       stepIndex,
		ModifiedPrompt:  modifiedPrompt,
	}, nil
}

// GetPromptEditRun returns the run linking the given edited trace to its
// original. Returns sql.ErrNoRows if no link exists.
func (db *DB) GetPromptEditRun(editedTraceID string) (PromptEditRun, error) {
	var run PromptEditRun
	err := db.conn.QueryRow(
		`SELECT id, original_trace_id, edited_trace_id, step_index, modified_prompt
		 FROM prompt_edit_runs
		 WHERE edited_trace_id = ?`,
		editedTraceID,
	).Scan(&run.ID, &run.OriginalTraceID, &run.EditedTraceID, &run.StepIndex, &run.ModifiedPrompt)
	if err != nil {
		return PromptEditRun{}, err
	}
	return run, nil
}
