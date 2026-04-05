package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recorded snapshots",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	store := snapshot.NewStore(cwd)

	snapshots, err := store.List()
	if err != nil {
		return fmt.Errorf("list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots recorded.")
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(snapshots, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal snapshots: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tSOURCE\tTIMESTAMP\tSTEPS")
	for _, s := range snapshots {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			s.Name,
			s.ID,
			s.Source,
			s.Timestamp.Format("2006-01-02 15:04:05"),
			len(s.Steps),
		)
	}
	return w.Flush()
}
