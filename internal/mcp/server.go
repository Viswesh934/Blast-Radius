package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/Viswesh934/blast-radius/pkg/netfetch"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

const protocolVersion = "2025-03-26"

type Server struct {
	codec     *stdioCodec
	cfg       *config.Config
	logger    *zap.Logger
	ingestion *IngestionPipeline
}

func NewServer(cfg *config.Config, logger *zap.Logger, ingestionDir string, in io.Reader, out io.Writer) (*Server, error) {
	ingestion, err := NewIngestionPipeline(ingestionDir)
	if err != nil {
		return nil, err
	}
	return &Server{
		codec:     newStdioCodec(in, out),
		cfg:       cfg,
		logger:    logger,
		ingestion: ingestion,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		payload, err := s.codec.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			_ = s.codec.writeMessage(jsonRPCResponse{JSONRPC: "2.0", Error: &jsonRPCError{Code: -32700, Message: "parse error"}})
			continue
		}
		if req.JSONRPC == "" {
			req.JSONRPC = "2.0"
		}

		if req.Method == "notifications/initialized" {
			continue
		}

		result, rpcErr := s.handleRequest(ctx, req)
		if req.ID == nil {
			continue
		}

		resp := jsonRPCResponse{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		if err := s.codec.writeMessage(resp); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req jsonRPCRequest) (any, *jsonRPCError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]string{
				"name":    "blast-radius-mcp",
				"version": "0.1.0",
			},
		}, nil
	case "ping":
		return map[string]any{"ok": true}, nil
	case "tools/list":
		return map[string]any{"tools": s.toolDefinitions()}, nil
	case "tools/call":
		res, err := s.handleToolCall(ctx, req.Params)
		if err != nil {
			return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": err.Error()}}}, nil
		}
		return map[string]any{"isError": false, "content": []map[string]string{{"type": "text", "text": res}}}, nil
	default:
		return nil, &jsonRPCError{Code: -32601, Message: "method not found"}
	}
}

func (s *Server) toolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "blast.snapshot.capture",
			"description": "Capture a fresh OpenMetadata snapshot for a source FQN.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source_fqn": map[string]any{"type": "string", "description": "Optional source FQN. Falls back to BR_DATABASE_FQN."},
				},
			},
		},
		{
			"name":        "blast.snapshot.compare",
			"description": "Compare two stored snapshot files.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"older_snapshot", "newer_snapshot"},
				"properties": map[string]any{
					"older_snapshot": map[string]any{"type": "string"},
					"newer_snapshot": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "blast.impact.analyze",
			"description": "Run impact analysis between two snapshot files.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"older_snapshot", "newer_snapshot"},
				"properties": map[string]any{
					"older_snapshot": map[string]any{"type": "string"},
					"newer_snapshot": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "blast.ingest.event",
			"description": "Push a real-time metadata event into Blast Radius ingestion queue.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"type", "source_fqn"},
				"properties": map[string]any{
					"id":         map[string]any{"type": "string"},
					"type":       map[string]any{"type": "string"},
					"source_fqn": map[string]any{"type": "string"},
					"entity_fqn": map[string]any{"type": "string"},
					"timestamp":  map[string]any{"type": "string", "description": "RFC3339 timestamp"},
					"payload":    map[string]any{"type": "object"},
				},
			},
		},
		{
			"name":        "blast.ingest.flush",
			"description": "Flush pending events for a source and generate snapshot+impact report.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source_fqn":        map[string]any{"type": "string"},
					"baseline_snapshot": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "blast.openmetadata.services.list",
			"description": "List available OpenMetadata database services for tool planning and parameter selection.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "blast.ingest.web.sync",
			"description": "Fetch public web JSON, infer fields, ingest to SQLite, sync OpenMetadata metadata, and run snapshot drift compare.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"url", "table", "service", "database", "schema"},
				"properties": map[string]any{
					"url":                  map[string]any{"type": "string"},
					"count":                map[string]any{"type": "integer", "description": "Record count to ingest. Defaults to 100."},
					"table":                map[string]any{"type": "string"},
					"service":              map[string]any{"type": "string"},
					"database":             map[string]any{"type": "string"},
					"schema":               map[string]any{"type": "string"},
					"db_file":              map[string]any{"type": "string", "description": "SQLite output file. Defaults to ./artifacts/ingested_web.db"},
					"description":          map[string]any{"type": "string", "description": "OpenMetadata table description."},
					"source_fqn":           map[string]any{"type": "string", "description": "Snapshot source FQN. Defaults to service.database.schema."},
					"allow_create_service": map[string]any{"type": "boolean", "description": "Whether the tool may create a missing OpenMetadata service."},
				},
			},
		},
	}
}

func (s *Server) handleToolCall(ctx context.Context, params json.RawMessage) (string, error) {
	var req struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return "", fmt.Errorf("parse tools/call params: %w", err)
	}

	switch req.Name {
	case "blast.snapshot.capture":
		source := stringArg(req.Arguments, "source_fqn")
		res, err := s.captureSnapshot(ctx, source)
		if err != nil {
			return "", err
		}
		return prettyJSON(res)
	case "blast.snapshot.compare":
		older := requiredStringArg(req.Arguments, "older_snapshot")
		if older == "" {
			return "", fmt.Errorf("older_snapshot is required")
		}
		newer := requiredStringArg(req.Arguments, "newer_snapshot")
		if newer == "" {
			return "", fmt.Errorf("newer_snapshot is required")
		}
		previous, err := snapshot.Load(older)
		if err != nil {
			return "", fmt.Errorf("load older snapshot: %w", err)
		}
		current, err := snapshot.Load(newer)
		if err != nil {
			return "", fmt.Errorf("load newer snapshot: %w", err)
		}
		diff := snapshot.CompareSnapshots(previous, current)
		return prettyJSON(map[string]any{"diff": diff})
	case "blast.impact.analyze":
		older := requiredStringArg(req.Arguments, "older_snapshot")
		if older == "" {
			return "", fmt.Errorf("older_snapshot is required")
		}
		newer := requiredStringArg(req.Arguments, "newer_snapshot")
		if newer == "" {
			return "", fmt.Errorf("newer_snapshot is required")
		}
		previous, err := snapshot.Load(older)
		if err != nil {
			return "", fmt.Errorf("load older snapshot: %w", err)
		}
		current, err := snapshot.Load(newer)
		if err != nil {
			return "", fmt.Errorf("load newer snapshot: %w", err)
		}
		diff := snapshot.CompareSnapshots(previous, current)
		analysis := impact.AnalyzeImpact(current, diff.Changes)
		return prettyJSON(map[string]any{"impact": analysis})
	case "blast.ingest.event":
		evt := IngestionEvent{
			ID:        stringArg(req.Arguments, "id"),
			Type:      requiredStringArg(req.Arguments, "type"),
			SourceFQN: requiredStringArg(req.Arguments, "source_fqn"),
			EntityFQN: stringArg(req.Arguments, "entity_fqn"),
			Payload:   objectArg(req.Arguments, "payload"),
		}
		if ts := stringArg(req.Arguments, "timestamp"); ts != "" {
			parsed, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return "", fmt.Errorf("timestamp must be RFC3339: %w", err)
			}
			evt.Timestamp = parsed.UTC()
		}
		accepted, err := s.ingestion.IngestEvent(evt)
		if err != nil {
			return "", err
		}
		return prettyJSON(map[string]any{
			"accepted":       accepted,
			"pending_events": s.ingestion.PendingCount(accepted.SourceFQN),
			"events_log":     filepath.Join(s.ingestion.baseDir, "events.ndjson"),
		})
	case "blast.ingest.flush":
		source := stringArg(req.Arguments, "source_fqn")
		baseline := stringArg(req.Arguments, "baseline_snapshot")
		result, err := s.ingestion.Flush(ctx, s.cfg, s.logger, source, baseline)
		if err != nil {
			return "", err
		}
		return prettyJSON(result)
	case "blast.openmetadata.services.list":
		client := openmetadata.NewClient(s.cfg.OpenMetadata.BaseURL, s.cfg.OpenMetadata.JWTToken, s.logger)
		services, err := client.ListDatabaseServices(ctx)
		if err != nil {
			return "", fmt.Errorf("list database services: %w", err)
		}
		return prettyJSON(map[string]any{
			"count":    len(services),
			"services": services,
		})
	case "blast.ingest.web.sync":
		result, err := s.ingestWebSync(ctx, req.Arguments)
		if err != nil {
			return "", err
		}
		return prettyJSON(result)
	default:
		return "", fmt.Errorf("unknown tool: %s", req.Name)
	}
}

func (s *Server) captureSnapshot(ctx context.Context, sourceFQN string) (map[string]any, error) {
	sourceFQN = strings.TrimSpace(sourceFQN)
	if sourceFQN == "" {
		sourceFQN = strings.TrimSpace(s.cfg.Database.FQN)
	}
	if sourceFQN == "" {
		return nil, fmt.Errorf("source_fqn is required either in call args or BR_DATABASE_FQN")
	}

	client := openmetadata.NewClient(s.cfg.OpenMetadata.BaseURL, s.cfg.OpenMetadata.JWTToken, s.logger)
	snap, err := snapshot.NewSnapshot(ctx, client, sourceFQN)
	if err != nil {
		return nil, fmt.Errorf("capture snapshot: %w", err)
	}

	sourceDir := filepath.Join(s.cfg.Snapshot.Directory, "sources", fqnToPath(sourceFQN))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return nil, fmt.Errorf("create source snapshot directory: %w", err)
	}
	path, err := snapshot.Save(snap, sourceDir)
	if err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}

	return map[string]any{
		"snapshot_id":   snap.ID,
		"source_fqn":    sourceFQN,
		"tables":        len(snap.Tables),
		"lineage_items": len(snap.Lineage),
		"snapshot_path": path,
	}, nil
}

func requiredStringArg(args map[string]any, key string) string {
	return strings.TrimSpace(stringArg(args, key))
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func objectArg(args map[string]any, key string) map[string]any {
	if args == nil {
		return nil
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return obj
}

func prettyJSON(payload any) (string, error) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json payload: %w", err)
	}
	return string(data), nil
}

type mcpInferredColumn struct {
	Name       string
	SQLiteType string
	OMType     string
}

func (s *Server) ingestWebSync(ctx context.Context, args map[string]any) (map[string]any, error) {
	url := requiredStringArg(args, "url")
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}
	table := requiredStringArg(args, "table")
	if table == "" {
		return nil, fmt.Errorf("table is required")
	}
	service := requiredStringArg(args, "service")
	if service == "" {
		return nil, fmt.Errorf("service is required")
	}
	database := requiredStringArg(args, "database")
	if database == "" {
		return nil, fmt.Errorf("database is required")
	}
	schemaName := requiredStringArg(args, "schema")
	if schemaName == "" {
		return nil, fmt.Errorf("schema is required")
	}

	count := intArg(args, "count", 100)
	if count <= 0 {
		return nil, fmt.Errorf("count must be a positive integer")
	}

	dbFile := strings.TrimSpace(stringArg(args, "db_file"))
	if dbFile == "" {
		dbFile = "./artifacts/ingested_web.db"
	}
	description := strings.TrimSpace(stringArg(args, "description"))
	if description == "" {
		description = "Ingested from public web JSON via MCP"
	}
	sourceFQN := strings.TrimSpace(stringArg(args, "source_fqn"))
	if sourceFQN == "" {
		sourceFQN = service + "." + database + "." + schemaName
	}
	allowCreateService := boolArg(args, "allow_create_service", false)

	records, err := mcpFetchPublicRecords(ctx, url, count)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no object records found from %s", url)
	}
	normalized := mcpNormalizeRecordKeys(records)
	cols := mcpInferSchema(normalized)
	if len(cols) == 0 {
		return nil, fmt.Errorf("could not infer schema from records")
	}

	ingestedRows, err := mcpIngestIntoSQLite(dbFile, table, cols, normalized)
	if err != nil {
		return nil, err
	}

	previousSnapshotPath, _ := s.latestSnapshotPathForSource(sourceFQN)

	client := openmetadata.NewClient(s.cfg.OpenMetadata.BaseURL, s.cfg.OpenMetadata.JWTToken, s.logger)
	if _, err := client.ListDatabaseServices(ctx); err != nil {
		return nil, fmt.Errorf("openmetadata preflight failed: %w", err)
	}

	databaseFQN, serviceCreated, err := s.ensureOMHierarchyForMCP(ctx, client, service, database, schemaName, allowCreateService)
	if err != nil {
		return nil, err
	}

	tableFQN, tableCreated, omWriteLimited, err := s.upsertOMTableForMCP(ctx, client, table, databaseFQN+"."+schemaName, description, cols)
	if err != nil {
		return nil, err
	}

	sampleDataStatus := "updated"
	samplePatch := mcpBuildSampleDataPatch(cols, normalized)
	if _, err := client.UpdateTable(ctx, tableFQN, "PATCH", samplePatch); err != nil {
		if mcpIsMethodNotAllowedError(err) {
			sampleDataStatus = "skipped_method_not_allowed"
		} else {
			return nil, fmt.Errorf("update table sample data: %w", err)
		}
	}

	newSnapshotPath, driftSummary, err := s.captureAndCompareSnapshotForMCP(ctx, sourceFQN)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"url":                url,
		"records_requested":  count,
		"records_ingested":   ingestedRows,
		"db_file":            dbFile,
		"table":              table,
		"source_fqn":         sourceFQN,
		"om_table_fqn":       tableFQN,
		"om_service_created": serviceCreated,
		"om_table_created":   tableCreated,
		"om_write_limited":   omWriteLimited,
		"sample_data":        sampleDataStatus,
		"previous_snapshot":  previousSnapshotPath,
		"new_snapshot":       newSnapshotPath,
		"drift":              driftSummary,
	}, nil
}

func (s *Server) ensureOMHierarchyForMCP(ctx context.Context, client *openmetadata.Client, service, database, schemaName string, allowCreateService bool) (string, bool, error) {
	serviceCreated := false
	if _, err := client.GetDatabaseServiceByName(ctx, service); err != nil {
		if !mcpIsNotFoundError(err) {
			return "", false, fmt.Errorf("lookup database service %q: %w", service, err)
		}
		available, _ := client.ListDatabaseServices(ctx)
		names := mcpRenderServiceNames(available)
		if !allowCreateService {
			if names == "" {
				return "", false, fmt.Errorf("openmetadata service %q is not available; set allow_create_service=true or use an existing service", service)
			}
			return "", false, fmt.Errorf("openmetadata service %q is not available; choose one of: %s", service, names)
		}

		payload := map[string]any{"name": service, "serviceType": "SQLite", "description": "Auto-created by blast.ingest.web.sync"}
		if _, createErr := client.CreateDatabaseService(ctx, payload); createErr != nil && !mcpIsAlreadyExistsError(createErr) {
			if !mcpIsMethodNotAllowedError(createErr) {
				return "", false, fmt.Errorf("create database service: %w", createErr)
			}
			if names == "" {
				return "", false, fmt.Errorf("openmetadata service %q is not available and create is not permitted", service)
			}
			return "", false, fmt.Errorf("openmetadata service %q is not available and create is not permitted; existing services: %s", service, names)
		}
		serviceCreated = true
	}

	if _, err := client.CreateDatabase(ctx, database, service, "Auto-created by blast.ingest.web.sync"); err != nil && !mcpIsAlreadyExistsError(err) {
		if !mcpIsMethodNotAllowedError(err) {
			return "", serviceCreated, fmt.Errorf("create database in openmetadata: %w", err)
		}
	}

	databaseFQN := strings.TrimSpace(service) + "." + strings.TrimSpace(database)
	if _, err := client.CreateDatabaseSchema(ctx, schemaName, databaseFQN, "Auto-created by blast.ingest.web.sync"); err != nil && !mcpIsAlreadyExistsError(err) {
		if !mcpIsMethodNotAllowedError(err) {
			return "", serviceCreated, fmt.Errorf("create schema in openmetadata: %w", err)
		}
	}

	return databaseFQN, serviceCreated, nil
}

func (s *Server) upsertOMTableForMCP(ctx context.Context, client *openmetadata.Client, tableName, schemaFQN, description string, cols []mcpInferredColumn) (string, bool, bool, error) {
	columns := make([]map[string]any, 0, len(cols))
	for _, c := range cols {
		columns = append(columns, map[string]any{
			"name":        c.Name,
			"dataType":    c.OMType,
			"description": "Auto-inferred from web JSON ingest via MCP",
		})
	}

	payload := map[string]any{
		"name":           mcpSanitizeIdentifier(tableName),
		"databaseSchema": schemaFQN,
		"description":    description,
		"columns":        columns,
	}

	table, err := client.CreateTable(ctx, tableName, schemaFQN, description, payload)
	if err == nil {
		return table.FullyQualifiedName, true, false, nil
	}

	fqn := schemaFQN + "." + mcpSanitizeIdentifier(tableName)
	if mcpIsMethodNotAllowedError(err) {
		existing, lookupErr := client.GetTableByName(ctx, fqn)
		if lookupErr != nil {
			return "", false, true, fmt.Errorf("create table method not allowed and existing table lookup failed: %w", lookupErr)
		}
		return existing.FullyQualifiedName, false, true, nil
	}
	if !mcpIsAlreadyExistsError(err) {
		return "", false, false, fmt.Errorf("create table in openmetadata: %w", err)
	}

	if _, err := client.UpdateTable(ctx, fqn, "PUT", payload); err != nil {
		if mcpIsMethodNotAllowedError(err) {
			existing, lookupErr := client.GetTableByName(ctx, fqn)
			if lookupErr != nil {
				return "", false, true, fmt.Errorf("update existing table method not allowed and lookup failed: %w", lookupErr)
			}
			return existing.FullyQualifiedName, false, true, nil
		}
		return "", false, false, fmt.Errorf("update existing table in openmetadata: %w", err)
	}

	return fqn, false, false, nil
}

func (s *Server) captureAndCompareSnapshotForMCP(ctx context.Context, sourceFQN string) (string, map[string]int, error) {
	client := openmetadata.NewClient(s.cfg.OpenMetadata.BaseURL, s.cfg.OpenMetadata.JWTToken, s.logger)
	current, err := snapshot.NewSnapshot(ctx, client, sourceFQN)
	if err != nil {
		return "", nil, fmt.Errorf("capture snapshot: %w", err)
	}

	sourceDir := filepath.Join(s.cfg.Snapshot.Directory, "sources", fqnToPath(sourceFQN))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create source snapshot directory: %w", err)
	}
	path, err := snapshot.Save(current, sourceDir)
	if err != nil {
		return "", nil, fmt.Errorf("save snapshot: %w", err)
	}

	previousPath, _ := mcpPreviousSnapshotPath(path, sourceDir)
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

func (s *Server) latestSnapshotPathForSource(sourceFQN string) (string, error) {
	sourceDir := filepath.Join(s.cfg.Snapshot.Directory, "sources", fqnToPath(sourceFQN))
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

func mcpPreviousSnapshotPath(currentPath, sourceDir string) (string, error) {
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

func mcpFetchPublicRecords(ctx context.Context, url string, maxRecords int) ([]map[string]any, error) {
	client := netfetch.NewClient(20 * time.Second)
	var payload any
	if _, err := client.GetJSON(ctx, strings.TrimSpace(url), map[string]string{"User-Agent": "blast-radius-mcp-web-ingest"}, nil, &payload); err != nil {
		return nil, fmt.Errorf("fetch records: %w", err)
	}

	all := mcpExtractRecordObjects(payload)
	if len(all) == 0 {
		return nil, nil
	}
	if maxRecords > len(all) {
		maxRecords = len(all)
	}
	return all[:maxRecords], nil
}

func mcpExtractRecordObjects(payload any) []map[string]any {
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

func mcpNormalizeRecordKeys(records []map[string]any) []map[string]any {
	normalized := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out := make(map[string]any, len(rec))
		for k, v := range rec {
			clean := mcpSanitizeIdentifier(k)
			if clean == "" {
				continue
			}
			out[clean] = v
		}
		normalized = append(normalized, out)
	}
	return normalized
}

func mcpInferSchema(records []map[string]any) []mcpInferredColumn {
	types := map[string]map[string]struct{}{}
	for _, rec := range records {
		for key, value := range rec {
			name := mcpSanitizeIdentifier(key)
			if name == "" {
				continue
			}
			if _, ok := types[name]; !ok {
				types[name] = map[string]struct{}{}
			}
			types[name][mcpClassifyType(value)] = struct{}{}
		}
	}

	cols := make([]mcpInferredColumn, 0, len(types))
	for name, candidates := range types {
		sqlType, omType := mcpMergeTypes(candidates)
		cols = append(cols, mcpInferredColumn{Name: name, SQLiteType: sqlType, OMType: omType})
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].Name < cols[j].Name })
	return cols
}

func mcpClassifyType(value any) string {
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

func mcpMergeTypes(candidates map[string]struct{}) (string, string) {
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

func mcpSanitizeIdentifier(value string) string {
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

func mcpIngestIntoSQLite(dbFile, tableName string, cols []mcpInferredColumn, records []map[string]any) (int, error) {
	table := mcpSanitizeIdentifier(tableName)
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

	columnDefs := make([]string, 0, len(cols))
	columnNames := make([]string, 0, len(cols))
	for _, col := range cols {
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
			values = append(values, mcpNormalizeDBValue(rec[colName]))
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

func mcpNormalizeDBValue(value any) any {
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

func mcpBuildSampleDataPatch(cols []mcpInferredColumn, records []map[string]any) []map[string]any {
	columnNames := make([]string, 0, len(cols))
	for _, c := range cols {
		columnNames = append(columnNames, c.Name)
	}

	rows := make([][]any, 0, len(records))
	for _, rec := range records {
		row := make([]any, 0, len(columnNames))
		for _, colName := range columnNames {
			row = append(row, mcpNormalizeDBValue(rec[colName]))
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

func mcpIsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=409") || strings.Contains(msg, "already exists") || strings.Contains(msg, "entity already exists")
}

func mcpIsMethodNotAllowedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=405") ||
		(strings.Contains(msg, "method") && strings.Contains(msg, "not supported")) ||
		strings.Contains(msg, "method not allowed")
}

func mcpIsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status=404") || strings.Contains(msg, "not found")
}

func mcpRenderServiceNames(services []openmetadata.DatabaseService) string {
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

func intArg(args map[string]any, key string, defaultVal int) int {
	if args == nil {
		return defaultVal
	}
	v, ok := args[key]
	if !ok || v == nil {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return defaultVal
	}
}

func boolArg(args map[string]any, key string, defaultVal bool) bool {
	if args == nil {
		return defaultVal
	}
	v, ok := args[key]
	if !ok || v == nil {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
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
