package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare <older-snapshot> <newer-snapshot>",
	Short: "Compare two snapshots",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		previous, err := snapshot.Load(args[0])
		if err != nil {
			return fmt.Errorf("load older snapshot: %w", err)
		}

		current, err := snapshot.Load(args[1])
		if err != nil {
			return fmt.Errorf("load newer snapshot: %w", err)
		}

		diff := snapshot.CompareSnapshots(previous, current)
		if strings.EqualFold(cfg.Output.Format, "json") {
			payload, err := json.MarshalIndent(diff, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal diff: %w", err)
			}
			fmt.Println(string(payload))
			return nil
		}

		fmt.Println("Change summary:")
		fmt.Printf("  added: %d\n", diff.Summary[string(snapshot.ChangeTypeAdded)])
		fmt.Printf("  deleted: %d\n", diff.Summary[string(snapshot.ChangeTypeDeleted)])
		fmt.Printf("  modified: %d\n", diff.Summary[string(snapshot.ChangeTypeModified)])
		fmt.Printf("  affected_tables: %d\n", len(diff.AffectedTables))

		if len(diff.Changes) == 0 {
			fmt.Println("No changes detected.")
			return nil
		}

		fmt.Println("Detailed changes:")
		for _, c := range diff.Changes {
			if c.Field == "" {
				fmt.Printf("  - [%s] %s severity=%s\n", c.Type, c.Entity, c.Severity)
				continue
			}
			fmt.Printf("  - [%s] %s field=%s old=%v new=%v severity=%s\n", c.Type, c.Entity, c.Field, c.OldValue, c.NewValue, c.Severity)
		}

		return nil
	},
}
