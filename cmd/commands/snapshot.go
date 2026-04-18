package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture current data state",
	Long:  "Create a snapshot of current schemas, tables, columns, and lineage.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, _ := zap.NewProduction()
		defer func() {
			_ = logger.Sync()
		}()

		if cfg.Database.FQN == "" {
			return fmt.Errorf("database.fqn is required (set BR_DATABASE_FQN or in config)")
		}

		if err := os.MkdirAll(cfg.Snapshot.Directory, 0o755); err != nil {
			return fmt.Errorf("create snapshot directory: %w", err)
		}

		client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
		snap, err := snapshot.NewSnapshot(context.Background(), client, cfg.Database.FQN)
		if err != nil {
			return fmt.Errorf("create snapshot: %w", err)
		}

		path, err := snapshot.Save(snap, cfg.Snapshot.Directory)
		if err != nil {
			return fmt.Errorf("save snapshot: %w", err)
		}

		fmt.Printf("snapshot_id: %s\n", snap.ID)
		fmt.Printf("tables: %d\n", len(snap.Tables))
		fmt.Printf("lineage_items: %d\n", len(snap.Lineage))
		fmt.Printf("file: %s\n", path)
		return nil
	},
}
