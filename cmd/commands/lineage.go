package commands

import (
	"fmt"

	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
)

var lineageCmd = &cobra.Command{
	Use:   "lineage <snapshot-file> <entity-fqn>",
	Short: "Inspect lineage for an entity from a snapshot",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := snapshot.Load(args[0])
		if err != nil {
			return fmt.Errorf("load snapshot: %w", err)
		}

		state, ok := snap.Lineage[args[1]]
		if !ok {
			fmt.Printf("No lineage entry found for %s\n", args[1])
			return nil
		}

		fmt.Printf("entity: %s\n", state.EntityFQN)
		fmt.Println("upstream:")
		for _, upstream := range state.UpstreamFQNs {
			fmt.Printf("  - %s\n", upstream)
		}
		fmt.Println("downstream:")
		for _, downstream := range state.DownstreamFQNs {
			fmt.Printf("  - %s\n", downstream)
		}
		return nil
	},
}
