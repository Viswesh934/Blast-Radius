package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	tableCreateName        string
	tableCreateSchemaFQN   string
	tableCreateDescription string
	tableCreatePayload     string
	tableCreatePayloadFile string

	tableUpdateMode string
	tableUpdateBody string
	tableUpdateFile string

	tableAddDataMode string
	tableAddDataBody string
	tableAddDataFile string
)

var tableCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a table in OpenMetadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		payload, err := loadJSONPayload(tableCreatePayload, tableCreatePayloadFile)
		if err != nil {
			return err
		}
		if payload == nil {
			if strings.TrimSpace(tableCreateName) == "" {
				return fmt.Errorf("--name is required unless --payload or --payload-file is provided")
			}
			if strings.TrimSpace(tableCreateSchemaFQN) == "" {
				return fmt.Errorf("--schema is required unless --payload or --payload-file is provided")
			}
			payload = map[string]any{
				"name":           strings.TrimSpace(tableCreateName),
				"databaseSchema": strings.TrimSpace(tableCreateSchemaFQN),
			}
			if strings.TrimSpace(tableCreateDescription) != "" {
				payload["description"] = strings.TrimSpace(tableCreateDescription)
			}
		}

		table, err := client.CreateTable(context.Background(), tableCreateName, tableCreateSchemaFQN, tableCreateDescription, payload)
		if err != nil {
			return fmt.Errorf("create table: %w", err)
		}

		if wantsJSONOutput() {
			return printStructured(map[string]any{"table": table})
		}

		fqn := table.FullyQualifiedName
		if fqn == "" {
			fqn = table.Name
		}
		fmt.Printf("created_table: %s\n", fqn)
		fmt.Printf("table_id: %s\n", table.ID)
		fmt.Printf("schema: %s\n", table.DatabaseSchema.FullyQualifiedName)
		return nil
	},
}

var tableGetCmd = &cobra.Command{
	Use:     "get <table-fqn>",
	Aliases: []string{"show", "describe", "inspect"},
	Short:   "Get a table by fully qualified name",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		table, err := client.GetTableByName(context.Background(), strings.TrimSpace(args[0]))
		if err != nil {
			return fmt.Errorf("get table: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"table": table})
		}

		fmt.Printf("table: %s\n", table.FullyQualifiedName)
		fmt.Printf("table_id: %s\n", table.ID)
		fmt.Printf("schema: %s\n", table.DatabaseSchema.FullyQualifiedName)
		fmt.Printf("description: %s\n", table.Description)
		return nil
	},
}

var tableDeleteCmd = &cobra.Command{
	Use:     "delete <table-fqn>",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a table by fully qualified name",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		tableFQN := strings.TrimSpace(args[0])
		if err := client.DeleteTable(context.Background(), tableFQN); err != nil {
			return fmt.Errorf("delete table: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"deleted_table": tableFQN})
		}

		fmt.Printf("deleted_table: %s\n", tableFQN)
		return nil
	},
}

var tableUpdateCmd = &cobra.Command{
	Use:     "update <table-fqn>",
	Aliases: []string{"edit", "patch"},
	Short:   "Update a table with a raw OpenMetadata payload",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		payload, err := loadJSONPayloadAny(tableUpdateBody, tableUpdateFile)
		if err != nil {
			return err
		}
		if payload == nil {
			return fmt.Errorf("--body or --body-file is required for table update")
		}

		updated, err := client.UpdateTable(context.Background(), strings.TrimSpace(args[0]), tableUpdateMode, payload)
		if err != nil {
			return fmt.Errorf("update table: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"table": updated})
		}

		fmt.Printf("updated_table: %s\n", updated.FullyQualifiedName)
		fmt.Printf("table_id: %s\n", updated.ID)
		return nil
	},
}

var tableAddDataCmd = &cobra.Command{
	Use:   "add-data <table-fqn>",
	Short: "Add sample data metadata to a table (OpenMetadata metadata, not DB row inserts)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		payload, err := loadJSONPayloadAny(tableAddDataBody, tableAddDataFile)
		if err != nil {
			return err
		}
		if payload == nil {
			return fmt.Errorf("--body or --body-file is required for add-data")
		}

		updated, err := client.UpdateTable(context.Background(), strings.TrimSpace(args[0]), tableAddDataMode, payload)
		if err != nil {
			return fmt.Errorf("add table sample data: %w", err)
		}

		if wantsJSONOutput() {
			return printStructured(map[string]any{"table": updated})
		}

		fmt.Printf("updated_table: %s\n", updated.FullyQualifiedName)
		fmt.Println("note: this updates OpenMetadata table metadata/sample data, not physical rows in the source database")
		return nil
	},
}

func init() {
	tableCreateCmd.Flags().StringVar(&tableCreateName, "name", "", "table name")
	tableCreateCmd.Flags().StringVar(&tableCreateSchemaFQN, "schema", "", "database schema FQN")
	tableCreateCmd.Flags().StringVar(&tableCreateDescription, "description", "", "table description")
	tableCreateCmd.Flags().StringVar(&tableCreatePayload, "payload", "", "raw JSON payload for /tables")
	tableCreateCmd.Flags().StringVar(&tableCreatePayloadFile, "payload-file", "", "path to JSON payload for /tables")

	tableUpdateCmd.Flags().StringVar(&tableUpdateMode, "method", "PATCH", "HTTP method for update (PATCH or PUT)")
	tableUpdateCmd.Flags().StringVar(&tableUpdateBody, "body", "", "raw JSON body for update")
	tableUpdateCmd.Flags().StringVar(&tableUpdateFile, "body-file", "", "path to JSON body file for update")

	tableAddDataCmd.Flags().StringVar(&tableAddDataMode, "method", "PATCH", "HTTP method for add-data (PATCH or PUT)")
	tableAddDataCmd.Flags().StringVar(&tableAddDataBody, "body", "", "raw JSON body for add-data")
	tableAddDataCmd.Flags().StringVar(&tableAddDataFile, "body-file", "", "path to JSON body file for add-data")

	tablesCmd.AddCommand(tableCreateCmd)
	tablesCmd.AddCommand(tableGetCmd)
	tablesCmd.AddCommand(tableDeleteCmd)
	tablesCmd.AddCommand(tableUpdateCmd)
	tablesCmd.AddCommand(tableAddDataCmd)
}
