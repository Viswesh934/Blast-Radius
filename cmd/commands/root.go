package commands

import (
	"fmt"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "blast-radius",
	Short: "Track data state changes and predict downstream impact",
	Long:  "Blast Radius captures database state snapshots and analyzes what changed and where it matters.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		loadedCfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg = loadedCfg
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file")
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(lineageCmd)
	rootCmd.AddCommand(validateCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
