package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/Viswesh934/blast-radius/internal/mcp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var mcpIngestionDir string

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run Blast Radius as an MCP server over stdio",
	Long:  "Starts a Model Context Protocol server that exposes Blast Radius snapshot, compare, impact, event ingestion, web ingestion sync, and service discovery tools.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, _ := zap.NewProduction()
		defer func() {
			_ = logger.Sync()
		}()

		server, err := mcp.NewServer(cfg, logger, mcpIngestionDir, os.Stdin, os.Stdout)
		if err != nil {
			return fmt.Errorf("create mcp server: %w", err)
		}

		if err := server.Run(context.Background()); err != nil {
			return fmt.Errorf("run mcp server: %w", err)
		}
		return nil
	},
}

func init() {
	mcpCmd.Flags().StringVar(&mcpIngestionDir, "ingestion-dir", "./snapshots/ingestion", "directory for ingestion events and reports")
}
