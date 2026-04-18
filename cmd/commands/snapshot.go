package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var snapshotSourceFQN string

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture a metadata snapshot from OpenMetadata",
	Long:  "Create a local snapshot of OpenMetadata tables, columns, and lineage for a database FQN.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, _ := zap.NewProduction()
		defer func() {
			_ = logger.Sync()
		}()

		sourceFQN := strings.TrimSpace(snapshotSourceFQN)
		if sourceFQN == "" {
			sourceFQN = strings.TrimSpace(cfg.Database.FQN)
		}
		if sourceFQN == "" {
			return fmt.Errorf("database.fqn is required (set BR_DATABASE_FQN, config, or pass --source)")
		}

		if err := os.MkdirAll(cfg.Snapshot.Directory, 0o755); err != nil {
			return fmt.Errorf("create snapshot directory: %w", err)
		}

		client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
		snap, err := snapshot.NewSnapshot(context.Background(), client, sourceFQN)
		if err != nil {
			return fmt.Errorf("create snapshot: %w", err)
		}

		sourceDir := filepath.Join(cfg.Snapshot.Directory, "sources", fqnToPath(sourceFQN))
		if err := os.MkdirAll(sourceDir, 0o755); err != nil {
			return fmt.Errorf("create source snapshot directory: %w", err)
		}

		path, err := snapshot.Save(snap, sourceDir)
		if err != nil {
			return fmt.Errorf("save snapshot: %w", err)
		}

		tableFiles := make([]string, 0, len(snap.Tables))
		tableFQNs := make([]string, 0, len(snap.Tables))
		for tableFQN := range snap.Tables {
			tableFQNs = append(tableFQNs, tableFQN)
		}
		sort.Strings(tableFQNs)

		for _, tableFQN := range tableFQNs {
			tableSnap := &snapshot.StateSnapshot{
				ID:        snap.ID,
				Timestamp: snap.Timestamp,
				Source:    tableFQN,
				Tables: map[string]*snapshot.TableState{
					tableFQN: snap.Tables[tableFQN],
				},
				Lineage:  map[string]*snapshot.LineageState{},
				Metadata: map[string]string{"version": "0.1.0", "parent_source": sourceFQN},
			}
			if lineage, ok := snap.Lineage[tableFQN]; ok {
				tableSnap.Lineage[tableFQN] = lineage
			}

			tableDir := filepath.Join(cfg.Snapshot.Directory, "tables", fqnToPath(tableFQN))
			if err := os.MkdirAll(tableDir, 0o755); err != nil {
				return fmt.Errorf("create table snapshot directory: %w", err)
			}
			tablePath, err := snapshot.Save(tableSnap, tableDir)
			if err != nil {
				return fmt.Errorf("save table snapshot for %s: %w", tableFQN, err)
			}
			tableFiles = append(tableFiles, tablePath)
		}

		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"snapshot_id":   snap.ID,
				"tables":        len(snap.Tables),
				"lineage_items": len(snap.Lineage),
				"file":          path,
				"table_files":   tableFiles,
				"source":        snap.Source,
				"snapshot":      snap,
			})
		}

		fmt.Printf("snapshot_id: %s\n", snap.ID)
		fmt.Printf("tables: %d\n", len(snap.Tables))
		fmt.Printf("lineage_items: %d\n", len(snap.Lineage))
		fmt.Printf("file: %s\n", path)
		fmt.Printf("table_snapshot_files: %d\n", len(tableFiles))
		return nil
	},
}

func init() {
	snapshotCmd.Flags().StringVar(&snapshotSourceFQN, "source", "", "database/schema FQN to snapshot (overrides BR_DATABASE_FQN)")
}

func fqnToPath(fqn string) string {
	fqn = strings.TrimSpace(fqn)
	if fqn == "" {
		return "unknown"
	}
	fqn = strings.ReplaceAll(fqn, "\\", "/")
	fqn = strings.ReplaceAll(fqn, "..", ".")
	parts := strings.Split(fqn, ".")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = strings.ReplaceAll(p, "/", "_")
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return "unknown"
	}
	return filepath.Join(clean...)
}
