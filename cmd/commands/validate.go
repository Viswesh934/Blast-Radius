package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate OpenMetadata connectivity and required configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(cfg.OpenMetadata.BaseURL) == "" {
			return fmt.Errorf("openmetadata.baseurl is required (set OM_BASE_URL or config)")
		}

		logger, _ := zap.NewProduction()
		defer func() {
			_ = logger.Sync()
		}()

		client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
		services, err := client.ListDatabaseServices(context.Background())
		if err != nil {
			return fmt.Errorf("openmetadata connectivity validation failed: %w", err)
		}
		result := map[string]any{
			"openmetadata_status":       "ok",
			"base_url":                  cfg.OpenMetadata.BaseURL,
			"database_services_visible": len(services),
		}

		databaseFQN := strings.TrimSpace(cfg.Database.FQN)
		if databaseFQN != "" {
			tables, err := client.GetTablesByDatabase(context.Background(), databaseFQN)
			if err != nil {
				return fmt.Errorf("openmetadata table visibility check failed for %s: %w", databaseFQN, err)
			}
			result["database_fqn"] = databaseFQN
			result["tables_visible"] = len(tables)
		} else {
			result["database_fqn"] = ""
			result["tables_visible"] = "skipped"
		}

		if wantsJSONOutput() {
			return printStructured(result)
		}
		fmt.Println("openmetadata_status: ok")
		fmt.Printf("base_url: %s\n", cfg.OpenMetadata.BaseURL)
		fmt.Printf("database_services_visible: %d\n", len(services))
		if databaseFQN != "" {
			fmt.Printf("database_fqn: %s\n", databaseFQN)
			fmt.Printf("tables_visible: %v\n", result["tables_visible"])
		} else {
			fmt.Println("database_fqn: (not set)")
			fmt.Println("tables_visible: skipped")
		}
		return nil
	},
}
