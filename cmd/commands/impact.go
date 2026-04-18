package commands

import (
	"fmt"

	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
)

var impactCmd = &cobra.Command{
	Use:   "impact <older-snapshot> <newer-snapshot>",
	Short: "Analyze downstream impact of changes",
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
		analysis := impact.AnalyzeImpact(current, diff.Changes)

		if wantsJSONOutput() {
			return printStructured(map[string]any{"impact": analysis})
		}

		fmt.Printf("risk_level: %s\n", analysis.RiskLevel)
		fmt.Printf("impacted_assets: %d\n", len(analysis.ImpactedAssets))
		for _, asset := range analysis.ImpactedAssets {
			fmt.Printf("  - %s (%s) reason=%s\n", asset.FQN, asset.Type, asset.Reason)
		}
		if len(analysis.RecommendedActions) > 0 {
			fmt.Println("recommended_actions:")
			for _, action := range analysis.RecommendedActions {
				fmt.Printf("  - %s\n", action)
			}
		}

		return nil
	},
}
