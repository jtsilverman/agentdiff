package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jtsilverman/agentdiff/internal/adapter"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	recordName        string
	recordAdapterName string
)

var recordCmd = &cobra.Command{
	Use:   "record [trace-file]",
	Short: "Record an agent trace as a snapshot",
	Long: `Record reads an agent execution trace and saves it as a structured snapshot.

The trace file can be a positional argument, "-" for stdin, or omitted to read stdin.
The adapter is auto-detected by default, or can be specified with --adapter.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRecord,
}

func init() {
	recordCmd.Flags().StringVar(&recordName, "name", "", "snapshot name (default: filename + timestamp)")
	recordCmd.Flags().StringVar(&recordAdapterName, "adapter", "auto", "adapter to use: claude, openai, or auto")
	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
	// Read input from file or stdin.
	var input []byte
	var sourceFile string
	var err error

	if len(args) == 0 || args[0] == "-" {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		sourceFile = "stdin"
	} else {
		sourceFile = args[0]
		input, err = os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	}

	// Select adapter.
	var a adapter.Adapter
	var sourceName string

	if recordAdapterName == "auto" {
		a, err = adapter.Detect(input)
		if err != nil {
			return fmt.Errorf("auto-detect adapter: %w", err)
		}
		sourceName = adapterSourceName(a)
	} else {
		a, err = adapter.Get(recordAdapterName)
		if err != nil {
			return err
		}
		sourceName = recordAdapterName
	}

	// Parse trace.
	steps, metadata, err := a.Parse(input)
	if err != nil {
		return fmt.Errorf("parse trace: %w", err)
	}

	// Generate name if not provided.
	name := recordName
	if name == "" {
		base := sourceFile
		if base != "stdin" {
			base = filepath.Base(base)
			ext := filepath.Ext(base)
			base = strings.TrimSuffix(base, ext)
		}
		name = base + "_" + time.Now().Format("20060102_150405")
	}

	// Create and save snapshot.
	snap := snapshot.Snapshot{
		Name:     name,
		Source:   sourceName,
		Metadata: metadata,
		Steps:    steps,
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	store := snapshot.NewStore(cwd)

	saved, err := store.Save(snap)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	fmt.Printf("Recorded snapshot: %s (%s)\n", saved.Name, saved.ID)
	return nil
}

// adapterSourceName returns a human-readable name for a detected adapter.
func adapterSourceName(a adapter.Adapter) string {
	switch a.(type) {
	case *adapter.ClaudeAdapter:
		return "claude"
	case *adapter.OpenAIAdapter:
		return "openai"
	default:
		return "unknown"
	}
}
