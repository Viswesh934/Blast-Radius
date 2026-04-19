package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"go.uber.org/zap"
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
