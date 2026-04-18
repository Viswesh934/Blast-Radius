package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var tablesCmd = &cobra.Command{
	Use:     "tables [database-fqn]",
	Aliases: []string{"table", "ls", "catalog"},
	Short:   "List tables from OpenMetadata for a database",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		databaseFQN := strings.TrimSpace(cfg.Database.FQN)
		if len(args) == 1 {
			databaseFQN = strings.TrimSpace(args[0])
		}
		if databaseFQN == "" {
			return fmt.Errorf("database FQN is required: pass argument or set BR_DATABASE_FQN")
		}
		if strings.TrimSpace(cfg.OpenMetadata.BaseURL) == "" {
			return fmt.Errorf("openmetadata.baseurl is required (set OM_BASE_URL or config)")
		}

		logger, _ := zap.NewProduction()
		defer func() {
			_ = logger.Sync()
		}()

		client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
		tables, err := client.GetTablesByDatabase(context.Background(), databaseFQN)
		if err != nil {
			return fmt.Errorf("list tables from openmetadata: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"database_fqn": databaseFQN,
				"table_count":  len(tables),
				"tables":       tables,
			})
		}

		fmt.Printf("database_fqn: %s\n", databaseFQN)
		fmt.Printf("table_count: %d\n", len(tables))
		for _, table := range tables {
			fmt.Printf("- %s\n", table.FQN)
		}
		return nil
	},
}
