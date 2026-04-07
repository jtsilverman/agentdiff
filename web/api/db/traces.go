package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// Trace represents a stored trace row.
type Trace struct {
	ID        string
	Name      string
	Adapter   string
	Source    string
	Metadata  map[string]string
	CreatedAt time.Time
}

// TraceSummary is a lightweight trace listing with step count.
type TraceSummary struct {
	ID        string
	Name      string
	Adapter   string
	StepCount int
	Metadata  map[string]string
	CreatedAt time.Time
}

// TraceDetail is a full trace with reconstructed steps.
type TraceDetail struct {
	ID        string
	Name      string
	Adapter   string
	Source    string
	Metadata  map[string]string
	Steps     []snapshot.Step
	CreatedAt time.Time
}

// CreateTrace inserts a new trace and returns it.
func (db *DB) CreateTrace(name, adapter string, metadata map[string]string) (Trace, error) {
	id := uuid.New().String()

	var metaJSON []byte
	var err error
	if metadata != nil {
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			return Trace{}, fmt.Errorf("marshal metadata: %w", err)
		}
	}

	var metaStr *string
	if metaJSON != nil {
		s := string(metaJSON)
		metaStr = &s
	}

	_, err = db.conn.Exec(
		`INSERT INTO traces (id, name, adapter, metadata) VALUES (?, ?, ?, ?)`,
		id, name, adapter, metaStr,
	)
	if err != nil {
		return Trace{}, fmt.Errorf("insert trace: %w", err)
	}

	return Trace{
		ID:        id,
		Name:      name,
		Adapter:   adapter,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}, nil
}

// ListTraces returns all traces with step counts.
func (db *DB) ListTraces() ([]TraceSummary, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.name, t.adapter,
			(SELECT COUNT(*) FROM snapshots s WHERE s.trace_id = t.id) AS step_count,
			t.metadata, t.created_at
		FROM traces t
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()

	var traces []TraceSummary
	for rows.Next() {
		var t TraceSummary
		var metaStr sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.Adapter, &t.StepCount, &metaStr, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan trace: %w", err)
		}
		if metaStr.Valid && metaStr.String != "" {
			if err := json.Unmarshal([]byte(metaStr.String), &t.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

// GetTrace returns a trace with all its steps reconstructed.
func (db *DB) GetTrace(id string) (TraceDetail, error) {
	var td TraceDetail
	var metaStr sql.NullString

	err := db.conn.QueryRow(
		`SELECT id, name, adapter, source, metadata, created_at FROM traces WHERE id = ?`, id,
	).Scan(&td.ID, &td.Name, &td.Adapter, &td.Source, &metaStr, &td.CreatedAt)
	if err != nil {
		return TraceDetail{}, fmt.Errorf("get trace: %w", err)
	}

	if metaStr.Valid && metaStr.String != "" {
		if err := json.Unmarshal([]byte(metaStr.String), &td.Metadata); err != nil {
			return TraceDetail{}, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	rows, err := db.conn.Query(`
		SELECT role, content, tool_name, tool_args, tool_output, tool_is_error
		FROM snapshots
		WHERE trace_id = ?
		ORDER BY step_index ASC
	`, id)
	if err != nil {
		return TraceDetail{}, fmt.Errorf("get trace steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return TraceDetail{}, err
		}
		td.Steps = append(td.Steps, step)
	}
	return td, rows.Err()
}

// scanStep reconstructs a snapshot.Step from a database row.
func scanStep(rows *sql.Rows) (snapshot.Step, error) {
	var (
		role        string
		content     sql.NullString
		toolName    sql.NullString
		toolArgs    sql.NullString
		toolOutput  sql.NullString
		toolIsError int
	)

	if err := rows.Scan(&role, &content, &toolName, &toolArgs, &toolOutput, &toolIsError); err != nil {
		return snapshot.Step{}, fmt.Errorf("scan step: %w", err)
	}

	step := snapshot.Step{
		Role:    role,
		Content: content.String,
	}

	// Rebuild ToolCall if tool_args is present (indicates a tool call step).
	if toolName.Valid && toolArgs.Valid && toolArgs.String != "" {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(toolArgs.String), &args); err != nil {
			return snapshot.Step{}, fmt.Errorf("unmarshal tool_args: %w", err)
		}
		step.ToolCall = &snapshot.ToolCall{
			Name: toolName.String,
			Args: args,
		}
	}

	// Rebuild ToolResult if tool_output is present (indicates a tool result step).
	if toolName.Valid && toolOutput.Valid {
		step.ToolResult = &snapshot.ToolResult{
			Name:    toolName.String,
			Output:  toolOutput.String,
			IsError: toolIsError != 0,
		}
	}

	return step, nil
}
