package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var databasesService string

var databasesCmd = &cobra.Command{
	Use:     "databases",
	Aliases: []string{"dbs"},
	Short:   "List OpenMetadata databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		dbs, err := client.ListDatabases(context.Background(), databasesService)
		if err != nil {
			return fmt.Errorf("list databases: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"database_count": len(dbs),
				"databases":      dbs,
			})
		}

		fmt.Printf("database_count: %d\n", len(dbs))
		for _, db := range dbs {
			fqn := db.FullyQualifiedName
			if fqn == "" {
				fqn = db.Name
			}
			svc := db.Service.FullyQualifiedName
			if svc == "" {
				svc = db.Service.Name
			}
			fmt.Printf("- %s (service=%s)\n", fqn, svc)
		}
		return nil
	},
}

func init() {
	databasesCmd.Flags().StringVar(&databasesService, "service", "", "filter databases by service name/FQN")
}
