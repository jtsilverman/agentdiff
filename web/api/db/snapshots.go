package db

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// InsertSnapshots batch-inserts steps into the snapshots table for a trace.
// Uses a transaction with a prepared statement for efficiency.
func (db *DB) InsertSnapshots(traceID string, steps []snapshot.Step) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO snapshots (id, trace_id, step_index, role, content, tool_name, tool_args, tool_output, tool_is_error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for i, step := range steps {
		var (
			toolName    *string
			toolArgs    *string
			toolOutput  *string
			toolIsError int
		)

		// NOTE: Each step should have either ToolCall or ToolResult, not both.
		// If both are set, ToolResult.Name overwrites ToolCall.Name in the DB row.
		if step.ToolCall != nil {
			toolName = &step.ToolCall.Name
			argsJSON, err := json.Marshal(step.ToolCall.Args)
			if err != nil {
				return fmt.Errorf("marshal tool args at step %d: %w", i, err)
			}
			s := string(argsJSON)
			toolArgs = &s
		}

		if step.ToolResult != nil {
			toolName = &step.ToolResult.Name
			toolOutput = &step.ToolResult.Output
			if step.ToolResult.IsError {
				toolIsError = 1
			}
		}

		_, err := stmt.Exec(
			uuid.New().String(),
			traceID,
			i,
			step.Role,
			step.Content,
			toolName,
			toolArgs,
			toolOutput,
			toolIsError,
		)
		if err != nil {
			return fmt.Errorf("insert step %d: %w", i, err)
		}
	}

	return tx.Commit()
}
