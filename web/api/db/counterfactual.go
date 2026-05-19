package db

import (
	"fmt"

	"github.com/google/uuid"
)

// CounterfactualRun is one persisted counterfactual replay linking an original trace
// to a re-run trace that diverged at step_index after modified_input was substituted.
type CounterfactualRun struct {
	ID                    string
	OriginalTraceID       string
	CounterfactualTraceID string
	StepIndex             int
	ModifiedInput         string
}

// CreateCounterfactualRun inserts a counterfactual_runs row and returns it.
// Returns an error wrapping FK violations when either trace ID is unknown.
func (db *DB) CreateCounterfactualRun(originalTraceID, counterfactualTraceID string, stepIndex int, modifiedInput string) (CounterfactualRun, error) {
	id := uuid.New().String()
	_, err := db.conn.Exec(
		`INSERT INTO counterfactual_runs (id, original_trace_id, counterfactual_trace_id, step_index, modified_input)
		 VALUES (?, ?, ?, ?, ?)`,
		id, originalTraceID, counterfactualTraceID, stepIndex, modifiedInput,
	)
	if err != nil {
		return CounterfactualRun{}, fmt.Errorf("insert counterfactual_run: %w", err)
	}
	return CounterfactualRun{
		ID:                    id,
		OriginalTraceID:       originalTraceID,
		CounterfactualTraceID: counterfactualTraceID,
		StepIndex:             stepIndex,
		ModifiedInput:         modifiedInput,
	}, nil
}

// GetCounterfactualRun returns the run linking the given counterfactual trace to its
// original. Returns sql.ErrNoRows if no link exists.
func (db *DB) GetCounterfactualRun(counterfactualTraceID string) (CounterfactualRun, error) {
	var run CounterfactualRun
	err := db.conn.QueryRow(
		`SELECT id, original_trace_id, counterfactual_trace_id, step_index, modified_input
		 FROM counterfactual_runs
		 WHERE counterfactual_trace_id = ?`,
		counterfactualTraceID,
	).Scan(&run.ID, &run.OriginalTraceID, &run.CounterfactualTraceID, &run.StepIndex, &run.ModifiedInput)
	if err != nil {
		return CounterfactualRun{}, err
	}
	return run, nil
}
