package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	createServiceName        string
	createServiceType        string
	createServiceDescription string
	createServicePayload     string
	createServicePayloadFile string

	createDatabaseName        string
	createDatabaseServiceName string
	createDatabaseDescription string

	createSchemaName        string
	createSchemaDatabaseFQN string
	createSchemaDescription string

	createTableName        string
	createTableSchemaFQN   string
	createTableDescription string
	createTablePayload     string
	createTablePayloadFile string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create OpenMetadata entities",
}

var createServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Create a database service in OpenMetadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		payload, err := loadJSONPayload(createServicePayload, createServicePayloadFile)
		if err != nil {
			return err
		}
		if payload == nil {
			if strings.TrimSpace(createServiceName) == "" {
				return fmt.Errorf("--name is required unless --payload or --payload-file is provided")
			}
			if strings.TrimSpace(createServiceType) == "" {
				return fmt.Errorf("--type is required unless --payload or --payload-file is provided")
			}
			payload = map[string]any{
				"name":        strings.TrimSpace(createServiceName),
				"serviceType": strings.TrimSpace(createServiceType),
			}
			if strings.TrimSpace(createServiceDescription) != "" {
				payload["description"] = strings.TrimSpace(createServiceDescription)
			}
		}

		svc, err := client.CreateDatabaseService(context.Background(), payload)
		if err != nil {
			return fmt.Errorf("create service: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"service": svc})
		}

		fqn := svc.FullyQualifiedName
		if fqn == "" {
			fqn = svc.Name
		}
		fmt.Printf("created_service: %s\n", fqn)
		fmt.Printf("service_type: %s\n", svc.ServiceType)
		fmt.Printf("service_id: %s\n", svc.ID)
		return nil
	},
}

var createDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Create a database in OpenMetadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(createDatabaseName) == "" {
			return fmt.Errorf("--name is required")
		}
		if strings.TrimSpace(createDatabaseServiceName) == "" {
			return fmt.Errorf("--service is required (database service name/FQN)")
		}

		client, err := newOMClient()
		if err != nil {
			return err
		}

		db, err := client.CreateDatabase(
			context.Background(),
			strings.TrimSpace(createDatabaseName),
			strings.TrimSpace(createDatabaseServiceName),
			strings.TrimSpace(createDatabaseDescription),
		)
		if err != nil {
			return fmt.Errorf("create database: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"database": db})
		}

		fqn := db.FullyQualifiedName
		if fqn == "" {
			fqn = db.Name
		}
		fmt.Printf("created_database: %s\n", fqn)
		fmt.Printf("database_id: %s\n", db.ID)
		fmt.Printf("service: %s\n", db.Service.FullyQualifiedName)
		return nil
	},
}

var createSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Create a database schema in OpenMetadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(createSchemaName) == "" {
			return fmt.Errorf("--name is required")
		}
		if strings.TrimSpace(createSchemaDatabaseFQN) == "" {
			return fmt.Errorf("--database is required (database FQN)")
		}

		client, err := newOMClient()
		if err != nil {
			return err
		}

		schema, err := client.CreateDatabaseSchema(
			context.Background(),
			strings.TrimSpace(createSchemaName),
			strings.TrimSpace(createSchemaDatabaseFQN),
			strings.TrimSpace(createSchemaDescription),
		)
		if err != nil {
			return fmt.Errorf("create schema: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"schema": schema})
		}

		fqn := schema.FullyQualifiedName
		if fqn == "" {
			fqn = schema.Name
		}
		fmt.Printf("created_schema: %s\n", fqn)
		fmt.Printf("schema_id: %s\n", schema.ID)
		fmt.Printf("database: %s\n", schema.Database.FullyQualifiedName)
		return nil
	},
}

var createTableCmd = &cobra.Command{
	Use:   "table",
	Short: "Create a table in OpenMetadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		payload, err := loadJSONPayload(createTablePayload, createTablePayloadFile)
		if err != nil {
			return err
		}
		if payload == nil {
			if strings.TrimSpace(createTableName) == "" {
				return fmt.Errorf("--name is required unless --payload or --payload-file is provided")
			}
			if strings.TrimSpace(createTableSchemaFQN) == "" {
				return fmt.Errorf("--schema is required unless --payload or --payload-file is provided")
			}
			payload = map[string]any{
				"name":           strings.TrimSpace(createTableName),
				"databaseSchema": strings.TrimSpace(createTableSchemaFQN),
			}
			if strings.TrimSpace(createTableDescription) != "" {
				payload["description"] = strings.TrimSpace(createTableDescription)
			}
		}

		table, err := client.CreateTable(context.Background(), createTableName, createTableSchemaFQN, createTableDescription, payload)
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

func init() {
	createServiceCmd.Flags().StringVar(&createServiceName, "name", "", "database service name")
	createServiceCmd.Flags().StringVar(&createServiceType, "type", "Postgres", "database service type (Postgres, MySQL, Snowflake, BigQuery, etc.)")
	createServiceCmd.Flags().StringVar(&createServiceDescription, "description", "", "service description")
	createServiceCmd.Flags().StringVar(&createServicePayload, "payload", "", "raw JSON payload for /services/databaseServices")
	createServiceCmd.Flags().StringVar(&createServicePayloadFile, "payload-file", "", "path to JSON payload for /services/databaseServices")

	createDatabaseCmd.Flags().StringVar(&createDatabaseName, "name", "", "database name")
	createDatabaseCmd.Flags().StringVar(&createDatabaseServiceName, "service", "", "database service name/FQN")
	createDatabaseCmd.Flags().StringVar(&createDatabaseDescription, "description", "", "database description")

	createSchemaCmd.Flags().StringVar(&createSchemaName, "name", "", "database schema name")
	createSchemaCmd.Flags().StringVar(&createSchemaDatabaseFQN, "database", "", "database FQN")
	createSchemaCmd.Flags().StringVar(&createSchemaDescription, "description", "", "schema description")

	createTableCmd.Flags().StringVar(&createTableName, "name", "", "table name")
	createTableCmd.Flags().StringVar(&createTableSchemaFQN, "schema", "", "database schema FQN")
	createTableCmd.Flags().StringVar(&createTableDescription, "description", "", "table description")
	createTableCmd.Flags().StringVar(&createTablePayload, "payload", "", "raw JSON payload for /tables")
	createTableCmd.Flags().StringVar(&createTablePayloadFile, "payload-file", "", "path to JSON payload for /tables")

	createCmd.AddCommand(createServiceCmd)
	createCmd.AddCommand(createDatabaseCmd)
	createCmd.AddCommand(createSchemaCmd)
	createCmd.AddCommand(createTableCmd)
}

func loadJSONPayload(raw, filePath string) (map[string]any, error) {
	payload := strings.TrimSpace(raw)
	if strings.TrimSpace(filePath) != "" {
		data, err := os.ReadFile(strings.TrimSpace(filePath))
		if err != nil {
			return nil, fmt.Errorf("read payload file: %w", err)
		}
		payload = strings.TrimSpace(string(data))
	}
	if payload == "" {
		return nil, nil
	}

	out := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil, fmt.Errorf("parse payload JSON: %w", err)
	}
	return out, nil
}

func loadJSONPayloadAny(raw, filePath string) (any, error) {
	payload := strings.TrimSpace(raw)
	if strings.TrimSpace(filePath) != "" {
		data, err := os.ReadFile(strings.TrimSpace(filePath))
		if err != nil {
			return nil, fmt.Errorf("read payload file: %w", err)
		}
		payload = strings.TrimSpace(string(data))
	}
	if payload == "" {
		return nil, nil
	}

	var out any
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil, fmt.Errorf("parse payload JSON: %w", err)
	}
	return out, nil
}
