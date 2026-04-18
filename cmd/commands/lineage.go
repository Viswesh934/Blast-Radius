package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var lineageCmd = &cobra.Command{
	Use:   "lineage <entity-fqn>",
	Short: "Fetch lineage for an entity from OpenMetadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(cfg.OpenMetadata.BaseURL) == "" {
			return fmt.Errorf("openmetadata.baseurl is required (set OM_BASE_URL or config)")
		}

		logger, _ := zap.NewProduction()
		defer func() {
			_ = logger.Sync()
		}()

		client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
		lineage, err := client.GetLineage(context.Background(), args[0], 10, 10)
		if err != nil {
			return fmt.Errorf("fetch lineage from openmetadata: %w", err)
		}

		if lineage == nil {
			if wantsJSONOutput() {
				return printStructured(map[string]any{
					"entity":     args[0],
					"upstream":   []string{},
					"downstream": []string{},
				})
			}
			fmt.Printf("No lineage found for %s\n", args[0])
			return nil
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"entity":     args[0],
				"upstream":   lineage.UpstreamFQNs,
				"downstream": lineage.DownstreamFQNs,
			})
		}

		fmt.Printf("entity: %s\n", args[0])
		fmt.Println("upstream:")
		for _, upstream := range lineage.UpstreamFQNs {
			fmt.Printf("  - %s\n", upstream)
		}
		fmt.Println("downstream:")
		for _, downstream := range lineage.DownstreamFQNs {
			fmt.Printf("  - %s\n", downstream)
		}
		return nil
	},
}
