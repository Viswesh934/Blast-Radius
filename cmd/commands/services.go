package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:     "services",
	Aliases: []string{"service"},
	Short:   "List OpenMetadata database services",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		services, err := client.ListDatabaseServices(context.Background())
		if err != nil {
			return fmt.Errorf("list database services: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"service_count": len(services),
				"services":      services,
			})
		}

		fmt.Printf("service_count: %d\n", len(services))
		for _, svc := range services {
			fqn := svc.FullyQualifiedName
			if fqn == "" {
				fqn = svc.Name
			}
			fmt.Printf("- %s (type=%s)\n", fqn, svc.ServiceType)
		}
		return nil
	},
}
