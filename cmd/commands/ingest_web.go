package commands

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/Viswesh934/blast-radius/pkg/netfetch"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

var (
	ingestWebURL          string
	ingestWebRecordCount  int
	ingestWebTableName    string
	ingestWebDBFile       string
	ingestWebOMService    string
	ingestWebOMDatabase   string
	ingestWebOMSchema     string
	ingestWebDescription  string
	ingestWebSourceFilter string
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingestion workflows",
}

var ingestWebCmd = &cobra.Command{
	Use:   "web",
	Short: "Fetch public JSON records, load them to DB, publish to OpenMetadata, and check drift",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(ingestWebURL) == "" {
			return fmt.Errorf("--url is required")
		}
		if strings.TrimSpace(ingestWebTableName) == "" {
			return fmt.Errorf("--table is required")
		}
		if strings.TrimSpace(ingestWebOMService) == "" {
			return fmt.Errorf("--service is required")
		}
		if strings.TrimSpace(ingestWebOMDatabase) == "" {
			return fmt.Errorf("--database is required")
		}
		if strings.TrimSpace(ingestWebOMSchema) == "" {
			return fmt.Errorf("--schema is required")
		}

		opts := ingestWebOptions{
			URL:         strings.TrimSpace(ingestWebURL),
			Count:       ingestWebRecordCount,
			Table:       strings.TrimSpace(ingestWebTableName),
			DBFile:      strings.TrimSpace(ingestWebDBFile),
			Service:     strings.TrimSpace(ingestWebOMService),
			Database:    strings.TrimSpace(ingestWebOMDatabase),
			Schema:      strings.TrimSpace(ingestWebOMSchema),
			Description: strings.TrimSpace(ingestWebDescription),
			SourceFQN:   strings.TrimSpace(ingestWebSourceFilter),
		}
		return executeWebIngest(opts)
	},
}

var ingestWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive guided web ingestion",
	Long:  "Guided mode that asks for website, record count, destination details, and then runs ingestion plus drift check.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Blast ingest wizard")
		fmt.Println("Answer a few questions to ingest data and check drift.")
		reader := bufio.NewReader(os.Stdin)

		suggestedTable := defaultTableNameFromURL(strings.TrimSpace(ingestWebURL))
		if suggestedTable == "" {
			suggestedTable = "web_records"
		}

		website, err := promptText(reader, "Website JSON URL", strings.TrimSpace(ingestWebURL), true)
		if err != nil {
			return err
		}
		count, err := promptInt(reader, "How many records should I ingest", ingestWebRecordCount, true)
		if err != nil {
			return err
		}
		table, err := promptText(reader, "Table name", firstNonEmpty(strings.TrimSpace(ingestWebTableName), suggestedTable), true)
		if err != nil {
			return err
		}
		service, err := promptText(reader, "OpenMetadata service", firstNonEmpty(strings.TrimSpace(ingestWebOMService), deriveServiceFromSource(cfg.Database.FQN), "web_service"), true)
		if err != nil {
			return err
		}
		database, err := promptText(reader, "OpenMetadata database", firstNonEmpty(strings.TrimSpace(ingestWebOMDatabase), "webdb"), true)
		if err != nil {
			return err
		}
		schema, err := promptText(reader, "OpenMetadata schema", firstNonEmpty(strings.TrimSpace(ingestWebOMSchema), "public"), true)
		if err != nil {
			return err
		}
		dbFile, err := promptText(reader, "Local SQLite file", firstNonEmpty(strings.TrimSpace(ingestWebDBFile), "./artifacts/ingested_web.db"), true)
		if err != nil {
			return err
		}
		description, err := promptText(reader, "Table description", firstNonEmpty(strings.TrimSpace(ingestWebDescription), "Ingested from public web JSON"), true)
		if err != nil {
			return err
		}
		defaultSource := firstNonEmpty(strings.TrimSpace(ingestWebSourceFilter), service+"."+database+"."+schema)
		source, err := promptText(reader, "Source FQN for snapshots", defaultSource, true)
		if err != nil {
			return err
		}

		opts := ingestWebOptions{
			URL:         website,
			Count:       count,
			Table:       table,
			DBFile:      dbFile,
			Service:     service,
			Database:    database,
			Schema:      schema,
			Description: description,
			SourceFQN:   source,
		}
		return executeWebIngest(opts)
	},
}

type ingestWebOptions struct {
	URL         string
	Count       int
	Table       string
	DBFile      string
	Service     string
	Database    string
	Schema      string
	Description string
	SourceFQN   string
}

func init() {
	ingestWebCmd.Flags().StringVar(&ingestWebURL, "url", "", "public JSON URL to ingest")
	ingestWebCmd.Flags().IntVar(&ingestWebRecordCount, "count", 0, "record count to ingest (if omitted, interactive prompt asks)")
	ingestWebCmd.Flags().StringVar(&ingestWebTableName, "table", "", "target table name")
	ingestWebCmd.Flags().StringVar(&ingestWebDBFile, "db-file", "./artifacts/ingested_web.db", "SQLite file path")
	ingestWebCmd.Flags().StringVar(&ingestWebOMService, "service", "", "OpenMetadata service name/FQN")
	ingestWebCmd.Flags().StringVar(&ingestWebOMDatabase, "database", "", "OpenMetadata database name")
	ingestWebCmd.Flags().StringVar(&ingestWebOMSchema, "schema", "", "OpenMetadata schema name")
	ingestWebCmd.Flags().StringVar(&ingestWebDescription, "description", "Ingested from public web JSON", "table description")
	ingestWebCmd.Flags().StringVar(&ingestWebSourceFilter, "source", "", "source FQN for snapshot capture and drift compare")

	ingestCmd.AddCommand(ingestWebCmd)
	ingestCmd.AddCommand(ingestWizardCmd)
}

func executeWebIngest(opts ingestWebOptions) error {
	count := opts.Count
	if count <= 0 {
		prompted, err := promptRecordCount()
		if err != nil {
			return err
		}
		count = prompted
	}

	fetchedRecords, err := fetchPublicRecords(opts.URL, count)
	if err != nil {
		return err
	}
	if len(fetchedRecords) == 0 {
		return fmt.Errorf("no object records found from %s", opts.URL)
	}
	normalizedRecords := normalizeRecordKeys(fetchedRecords)

	schema := inferSchema(normalizedRecords)
	if len(schema) == 0 {
		return fmt.Errorf("could not infer schema from records")
	}

	dbFile := opts.DBFile
	if dbFile == "" {
		dbFile = "./artifacts/ingested_web.db"
	}
	ingestedRows, err := ingestIntoSQLite(dbFile, opts.Table, schema, normalizedRecords)
	if err != nil {
		return err
	}

	sourceFQN := opts.SourceFQN
	if sourceFQN == "" {
		sourceFQN = strings.TrimSpace(cfg.Database.FQN)
	}
	if sourceFQN == "" {
		sourceFQN = opts.Service + "." + opts.Database + "." + opts.Schema
	}

	previousSnapshotPath, _ := latestSnapshotPath(cfg.Snapshot.Directory, sourceFQN)

	omClient, err := newOMClient()
	if err != nil {
		return err
	}
	if err := validateOpenMetadataClient(omClient); err != nil {
		return err
	}

	databaseFQN, err := ensureOMHierarchy(omClient, opts.Service, opts.Database, opts.Schema)
	if err != nil {
		return err
	}

	ingestWebDescription = opts.Description
	tableFQN, created, omWriteLimited, err := upsertOMTable(omClient, opts.Table, databaseFQN+"."+opts.Schema, schema)
	if err != nil {
		return err
	}

	samplePatch := buildSampleDataPatch(schema, normalizedRecords)
	sampleDataStatus := "updated"
	if _, err := omClient.UpdateTable(context.Background(), tableFQN, "PATCH", samplePatch); err != nil {
		if isMethodNotAllowedError(err) {
			sampleDataStatus = "skipped_method_not_allowed"
		} else {
			return fmt.Errorf("update table sample data: %w", err)
		}
	}

	newSnapshotPath, driftSummary, err := captureAndCompareSnapshot(sourceFQN)
	if err != nil {
		return err
	}

	if wantsJSONOutput() {
		return printStructured(map[string]any{
			"ingest": map[string]any{
				"url":               opts.URL,
				"records_requested": count,
				"records_ingested":  ingestedRows,
				"db_file":           dbFile,
				"table":             opts.Table,
				"source_fqn":        sourceFQN,
				"om_table_fqn":      tableFQN,
				"om_table_created":  created,
				"om_write_limited":  omWriteLimited,
				"sample_data":       sampleDataStatus,
				"previous_snapshot": previousSnapshotPath,
				"new_snapshot":      newSnapshotPath,
				"drift":             driftSummary,
			},
		})
	}

	fmt.Println("web ingestion completed")
	fmt.Printf("url: %s\n", opts.URL)
	fmt.Printf("records_requested: %d\n", count)
	fmt.Printf("records_ingested: %d\n", ingestedRows)
	fmt.Printf("db_file: %s\n", dbFile)
	fmt.Printf("openmetadata_table: %s\n", tableFQN)
	if created {
		fmt.Println("openmetadata_table_status: created")
	} else {
		fmt.Println("openmetadata_table_status: updated")
	}
	if omWriteLimited {
		fmt.Println("openmetadata_write_mode: limited (read-only or restricted methods)")
	}
	fmt.Printf("sample_data_status: %s\n", sampleDataStatus)
	if previousSnapshotPath == "" {
		fmt.Println("drift_status: baseline_created (no previous snapshot found)")
	} else {
		fmt.Println("drift_status: compared")
		fmt.Printf("previous_snapshot: %s\n", previousSnapshotPath)
		fmt.Printf("new_snapshot: %s\n", newSnapshotPath)
		fmt.Printf("drift_added: %d\n", driftSummary[string(snapshot.ChangeTypeAdded)])
		fmt.Printf("drift_deleted: %d\n", driftSummary[string(snapshot.ChangeTypeDeleted)])
		fmt.Printf("drift_modified: %d\n", driftSummary[string(snapshot.ChangeTypeModified)])
	}

	return nil
}

func promptRecordCount() (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("How many records should I ingest? ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("read record count: %w", err)
	}
	value, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("record count must be a positive integer")
	}
	return value, nil
}

func promptText(reader *bufio.Reader, label, defaultValue string, required bool) (string, error) {
	if strings.TrimSpace(defaultValue) != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = strings.TrimSpace(defaultValue)
	}
	if required && value == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return value, nil
}

func promptInt(reader *bufio.Reader, label string, defaultValue int, required bool) (int, error) {
	if defaultValue > 0 {
		fmt.Printf("%s [%d]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", strings.ToLower(label), err)
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		if defaultValue > 0 {
			return defaultValue, nil
		}
		if required {
			return 0, fmt.Errorf("%s is required", strings.ToLower(label))
		}
		return 0, nil
	}
	value, convErr := strconv.Atoi(trimmed)
	if convErr != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", strings.ToLower(label))
	}
	return value, nil
}

func defaultTableNameFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return ""
	}
	segments := strings.Split(path, "/")
	return sanitizeIdentifier(segments[len(segments)-1])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func deriveServiceFromSource(sourceFQN string) string {
	parts := strings.Split(strings.TrimSpace(sourceFQN), ".")
	if len(parts) == 0 {
		return ""
	}
	return sanitizeIdentifier(parts[0])
}

func fetchPublicRecords(url string, maxRecords int) ([]map[string]any, error) {
	client := netfetch.NewClient(20 * time.Second)
	var payload any
	if _, err := client.GetJSON(context.Background(), strings.TrimSpace(url), map[string]string{"User-Agent": "blast-radius-web-ingest"}, nil, &payload); err != nil {
		return nil, fmt.Errorf("fetch records: %w", err)
	}

	all := extractRecordObjects(payload)
	if len(all) == 0 {
		return nil, nil
	}
	if maxRecords > len(all) {
		maxRecords = len(all)
	}
	return all[:maxRecords], nil
}

func extractRecordObjects(payload any) []map[string]any {
	toObjects := func(items []any) []map[string]any {
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			obj, ok := item.(map[string]any)
			if ok {
				out = append(out, obj)
			}
		}
		return out
	}

	switch v := payload.(type) {
	case []any:
		return toObjects(v)
	case map[string]any:
		for _, key := range []string{"data", "results", "items", "records"} {
			candidate, ok := v[key]
			if !ok {
				continue
			}
			if arr, ok := candidate.([]any); ok {
				return toObjects(arr)
			}
		}
	}
	return nil
}

func normalizeRecordKeys(records []map[string]any) []map[string]any {
	normalized := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out := make(map[string]any, len(rec))
		for k, v := range rec {
			clean := sanitizeIdentifier(k)
			if clean == "" {
				continue
			}
			out[clean] = v
		}
		normalized = append(normalized, out)
	}
	return normalized
}

type inferredColumn struct {
	Name       string
	SQLiteType string
	OMType     string
}

func inferSchema(records []map[string]any) []inferredColumn {
	types := map[string]map[string]struct{}{}
	for _, rec := range records {
		for key, value := range rec {
			colName := sanitizeIdentifier(key)
			if colName == "" {
				continue
			}
			if _, ok := types[colName]; !ok {
				types[colName] = map[string]struct{}{}
			}
			types[colName][classifyType(value)] = struct{}{}
		}
	}

	cols := make([]inferredColumn, 0, len(types))
	for name, candidates := range types {
		sqlType, omType := mergeTypes(candidates)
		cols = append(cols, inferredColumn{Name: name, SQLiteType: sqlType, OMType: omType})
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].Name < cols[j].Name })
	return cols
}

func classifyType(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case map[string]any, []any:
		return "json"
	case nil:
		return "null"
	default:
		return "string"
	}
}

func mergeTypes(candidates map[string]struct{}) (string, string) {
	if len(candidates) == 1 {
		if _, ok := candidates["bool"]; ok {
			return "INTEGER", "BOOLEAN"
		}
		if _, ok := candidates["number"]; ok {
			return "REAL", "DOUBLE"
		}
		if _, ok := candidates["string"]; ok {
			return "TEXT", "STRING"
		}
		if _, ok := candidates["json"]; ok {
			return "TEXT", "STRING"
		}
	}
	if _, hasString := candidates["string"]; hasString {
		return "TEXT", "STRING"
	}
	if _, hasJSON := candidates["json"]; hasJSON {
		return "TEXT", "STRING"
	}
	if _, hasNumber := candidates["number"]; hasNumber {
		return "REAL", "DOUBLE"
	}
	if _, hasBool := candidates["bool"]; hasBool {
		return "INTEGER", "BOOLEAN"
	}
	return "TEXT", "STRING"
}

func sanitizeIdentifier(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	b := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "field"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "f_" + out
	}
	return out
}

func ingestIntoSQLite(dbFile, tableName string, schema []inferredColumn, records []map[string]any) (int, error) {
	table := sanitizeIdentifier(tableName)
	if table == "" {
		return 0, fmt.Errorf("invalid table name")
	}
	if err := os.MkdirAll(filepath.Dir(dbFile), 0o755); err != nil {
		return 0, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return 0, fmt.Errorf("open sqlite db: %w", err)
	}
	defer func() { _ = db.Close() }()

	columnDefs := make([]string, 0, len(schema))
	columnNames := make([]string, 0, len(schema))
	for _, col := range schema {
		columnDefs = append(columnDefs, col.Name+" "+col.SQLiteType)
		columnNames = append(columnNames, col.Name)
	}

	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, strings.Join(columnDefs, ", "))
	if _, err := db.Exec(createSQL); err != nil {
		return 0, fmt.Errorf("create table in sqlite: %w", err)
	}

	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columnNames, ", "), strings.Join(placeholders, ", "))

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("start sqlite tx: %w", err)
	}
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare sqlite insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	count := 0
	for _, rec := range records {
		values := make([]any, 0, len(columnNames))
		for _, colName := range columnNames {
			values = append(values, normalizeDBValue(rec[colName]))
		}
		if _, err := stmt.Exec(values...); err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("insert row into sqlite: %w", err)
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit sqlite tx: %w", err)
	}

	return count, nil
}

func normalizeDBValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case bool:
		if v {
			return 1
		}
		return 0
	case float64, string:
		return v
	case map[string]any, []any:
		data, _ := json.Marshal(v)
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func ensureOMHierarchy(client *openmetadata.Client, serviceName, databaseName, schemaName string) (string, error) {
	ctx := context.Background()
	exists, err := serviceExists(client, serviceName)
	if err != nil {
		return "", err
	}
	if !exists {
		services, listErr := client.ListDatabaseServices(ctx)
		if listErr != nil {
			services = nil
		}
		existingNames := renderServiceNames(services)
		if !canPromptStdin() {
			if existingNames == "" {
				return "", fmt.Errorf("openmetadata service %q is not available; no existing services were discoverable and create permission may be required", serviceName)
			}
			return "", fmt.Errorf("openmetadata service %q is not available; choose one of: %s", serviceName, existingNames)
		}

		fmt.Printf("OpenMetadata service %q was not found.\n", serviceName)
		if existingNames != "" {
			fmt.Printf("Existing services: %s\n", existingNames)
		}
		createNow, promptErr := promptYesNo("Create this service now", false)
		if promptErr != nil {
			return "", promptErr
		}
		if !createNow {
			if existingNames == "" {
				return "", fmt.Errorf("openmetadata service %q is not available; choose an existing --service or allow create", serviceName)
			}
			return "", fmt.Errorf("openmetadata service %q is not available; choose one of: %s", serviceName, existingNames)
		}

		servicePayload := map[string]any{"name": serviceName, "serviceType": "SQLite", "description": "Auto-created by blast ingest web"}
		if _, createErr := client.CreateDatabaseService(ctx, servicePayload); createErr != nil && !isAlreadyExistsError(createErr) {
			if !isMethodNotAllowedError(createErr) {
				return "", fmt.Errorf("create service in openmetadata: %w", createErr)
			}
			if existingNames == "" {
				return "", fmt.Errorf("openmetadata service %q is not available; create is not permitted in this environment", serviceName)
			}
			return "", fmt.Errorf("openmetadata service %q is not available; create is not permitted. existing services: %s", serviceName, existingNames)
		}

		exists, err = serviceExists(client, serviceName)
		if err != nil {
			return "", err
		}
		if !exists {
			if existingNames == "" {
				return "", fmt.Errorf("openmetadata service %q is not available after create attempt", serviceName)
			}
			return "", fmt.Errorf("openmetadata service %q is not available; choose one of: %s", serviceName, existingNames)
		}
	}

	if _, err := client.CreateDatabase(ctx, databaseName, serviceName, "Auto-created by blast ingest web"); err != nil && !isAlreadyExistsError(err) {
		if !isMethodNotAllowedError(err) {
			return "", fmt.Errorf("create database in openmetadata: %w", err)
		}
	}

	databaseFQN := strings.TrimSpace(serviceName) + "." + strings.TrimSpace(databaseName)
	if _, err := client.CreateDatabaseSchema(ctx, schemaName, databaseFQN, "Auto-created by blast ingest web"); err != nil && !isAlreadyExistsError(err) {
		if !isMethodNotAllowedError(err) {
			return "", fmt.Errorf("create schema in openmetadata: %w", err)
		}
	}

	return databaseFQN, nil
}

func upsertOMTable(client *openmetadata.Client, tableName, schemaFQN string, cols []inferredColumn) (string, bool, bool, error) {
	columns := make([]map[string]any, 0, len(cols))
	for _, c := range cols {
		columns = append(columns, map[string]any{
			"name":        c.Name,
			"dataType":    c.OMType,
			"description": "Auto-inferred from web JSON ingest",
		})
	}

	payload := map[string]any{
		"name":           sanitizeIdentifier(tableName),
		"databaseSchema": schemaFQN,
		"description":    ingestWebDescription,
		"columns":        columns,
	}

	table, err := client.CreateTable(context.Background(), tableName, schemaFQN, ingestWebDescription, payload)
	if err == nil {
		return table.FullyQualifiedName, true, false, nil
	}

	fqn := schemaFQN + "." + sanitizeIdentifier(tableName)
	if isMethodNotAllowedError(err) {
		existing, lookupErr := client.GetTableByName(context.Background(), fqn)
		if lookupErr != nil {
			return "", false, true, fmt.Errorf("create table in openmetadata is not allowed and existing table lookup failed: %w", lookupErr)
		}
		return existing.FullyQualifiedName, false, true, nil
	}
	if !isAlreadyExistsError(err) {
		return "", false, false, fmt.Errorf("create table in openmetadata: %w", err)
	}

	if _, err := client.UpdateTable(context.Background(), fqn, "PUT", payload); err != nil {
		if isMethodNotAllowedError(err) {
			existing, lookupErr := client.GetTableByName(context.Background(), fqn)
			if lookupErr != nil {
				return "", false, true, fmt.Errorf("update existing table in openmetadata is not allowed and existing table lookup failed: %w", lookupErr)
			}
			return existing.FullyQualifiedName, false, true, nil
		}
		return "", false, false, fmt.Errorf("update existing table in openmetadata: %w", err)
	}
	return fqn, false, false, nil
}

func buildSampleDataPatch(cols []inferredColumn, records []map[string]any) []map[string]any {
	columnNames := make([]string, 0, len(cols))
	for _, c := range cols {
		columnNames = append(columnNames, c.Name)
	}

	rows := make([][]any, 0, len(records))
	for _, rec := range records {
		row := make([]any, 0, len(columnNames))
		for _, colName := range columnNames {
			row = append(row, normalizeDBValue(rec[colName]))
		}
		rows = append(rows, row)
	}

	return []map[string]any{
		{
			"op":   "add",
			"path": "/sampleData",
			"value": map[string]any{
				"columns": columnNames,
				"rows":    rows,
			},
		},
	}
}

func captureAndCompareSnapshot(sourceFQN string) (string, map[string]int, error) {
	logger, _ := zap.NewProduction()
	defer func() {
		_ = logger.Sync()
	}()

	if err := os.MkdirAll(cfg.Snapshot.Directory, 0o755); err != nil {
		return "", nil, fmt.Errorf("create snapshot directory: %w", err)
	}

	client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
	current, err := snapshot.NewSnapshot(context.Background(), client, sourceFQN)
	if err != nil {
		return "", nil, fmt.Errorf("capture snapshot: %w", err)
	}

	sourceDir := filepath.Join(cfg.Snapshot.Directory, "sources", fqnToPath(sourceFQN))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create source snapshot directory: %w", err)
	}

	path, err := snapshot.Save(current, sourceDir)
	if err != nil {
		return "", nil, fmt.Errorf("save snapshot: %w", err)
	}

	previousPath, _ := previousSnapshotPath(path, sourceDir)
	if previousPath == "" {
		return path, map[string]int{
			string(snapshot.ChangeTypeAdded):    0,
			string(snapshot.ChangeTypeDeleted):  0,
			string(snapshot.ChangeTypeModified): 0,
		}, nil
	}

	previous, err := snapshot.Load(previousPath)
	if err != nil {
		return "", nil, fmt.Errorf("load previous snapshot: %w", err)
	}
	diff := snapshot.CompareSnapshots(previous, current)
	return path, diff.Summary, nil
}

func previousSnapshotPath(currentPath, sourceDir string) (string, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return "", err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "snapshot_") && strings.HasSuffix(name, ".json") {
			paths = append(paths, filepath.Join(sourceDir, name))
		}
	}
	sort.Strings(paths)
	if len(paths) < 2 {
		return "", nil
	}
	for i := len(paths) - 1; i >= 0; i-- {
		if paths[i] != currentPath {
			return paths[i], nil
		}
	}
	return "", nil
}

func latestSnapshotPath(snapshotDir, sourceFQN string) (string, error) {
	sourceDir := filepath.Join(snapshotDir, "sources", fqnToPath(sourceFQN))
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return "", err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "snapshot_") && strings.HasSuffix(name, ".json") {
			paths = append(paths, filepath.Join(sourceDir, name))
		}
	}
	if len(paths) == 0 {
		return "", nil
	}
	sort.Strings(paths)
	return paths[len(paths)-1], nil
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=409") || strings.Contains(msg, "already exists") || strings.Contains(msg, "entity already exists")
}

func isMethodNotAllowedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=405") ||
		(strings.Contains(msg, "method") && strings.Contains(msg, "not supported")) ||
		strings.Contains(msg, "method not allowed")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=404") || strings.Contains(msg, "not found")
}

func serviceExists(client *openmetadata.Client, serviceName string) (bool, error) {
	_, err := client.GetDatabaseServiceByName(context.Background(), strings.TrimSpace(serviceName))
	if err == nil {
		return true, nil
	}
	if isNotFoundError(err) {
		return false, nil
	}
	return false, fmt.Errorf("lookup database service name=%s failed: %w", strings.TrimSpace(serviceName), err)
}

func renderServiceNames(services []openmetadata.DatabaseService) string {
	if len(services) == 0 {
		return ""
	}
	names := make([]string, 0, len(services))
	for _, svc := range services {
		name := strings.TrimSpace(svc.Name)
		if name == "" {
			name = strings.TrimSpace(svc.FullyQualifiedName)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func canPromptStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func promptYesNo(question string, defaultYes bool) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	if defaultYes {
		fmt.Printf("%s [Y/n]: ", question)
	} else {
		fmt.Printf("%s [y/N]: ", question)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	choice := strings.ToLower(strings.TrimSpace(line))
	if choice == "" {
		return defaultYes, nil
	}
	if choice == "y" || choice == "yes" {
		return true, nil
	}
	if choice == "n" || choice == "no" {
		return false, nil
	}
	return false, fmt.Errorf("please answer yes or no")
}

func validateOpenMetadataClient(client *openmetadata.Client) error {
	if _, err := client.ListDatabaseServices(context.Background()); err != nil {
		return fmt.Errorf("openmetadata preflight failed: %w (check OM_BASE_URL includes /api/v1 and credentials are valid)", err)
	}
	return nil
}
