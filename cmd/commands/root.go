package commands

import (
	"fmt"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfg          *config.Config
	cfgFile      string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:     "blast",
	Aliases: []string{"blast-radius", "br"},
	Short:   "OpenMetadata-first CLI for metadata snapshots and impact analysis",
	Long:    "Blast Radius connects to OpenMetadata to snapshot metadata, inspect lineage, compare changes, and evaluate downstream impact.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		loadedCfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if outputFormat != "" {
			loadedCfg.Output.Format = outputFormat
		}
		cfg = loadedCfg
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "", "output format override (table or json)")
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(tablesCmd)
	rootCmd.AddCommand(servicesCmd)
	rootCmd.AddCommand(databasesCmd)
	rootCmd.AddCommand(glossaryCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(guardCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(lineageCmd)
	rootCmd.AddCommand(validateCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
