package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/openmetadata"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"go.uber.org/zap"
)

type IngestionEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	SourceFQN string         `json:"source_fqn"`
	EntityFQN string         `json:"entity_fqn,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type IngestionPipeline struct {
	baseDir   string
	eventsLog string

	mu      sync.Mutex
	pending map[string][]IngestionEvent
}

func NewIngestionPipeline(baseDir string) (*IngestionPipeline, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = filepath.Join(".", "snapshots", "ingestion")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create ingestion directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(baseDir, "reports"), 0o755); err != nil {
		return nil, fmt.Errorf("create ingestion report directory: %w", err)
	}

	p := &IngestionPipeline{
		baseDir:   baseDir,
		eventsLog: filepath.Join(baseDir, "events.ndjson"),
		pending:   make(map[string][]IngestionEvent),
	}
	return p, nil
}

func (p *IngestionPipeline) IngestEvent(event IngestionEvent) (IngestionEvent, error) {
	event.Type = strings.TrimSpace(event.Type)
	event.SourceFQN = strings.TrimSpace(event.SourceFQN)
	event.EntityFQN = strings.TrimSpace(event.EntityFQN)
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Type == "" {
		return IngestionEvent{}, fmt.Errorf("event.type is required")
	}
	if event.SourceFQN == "" {
		return IngestionEvent{}, fmt.Errorf("event.source_fqn is required")
	}

	line, err := json.Marshal(event)
	if err != nil {
		return IngestionEvent{}, fmt.Errorf("marshal event: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := os.OpenFile(p.eventsLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return IngestionEvent{}, fmt.Errorf("open events log: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return IngestionEvent{}, fmt.Errorf("append event: %w", err)
	}

	p.pending[event.SourceFQN] = append(p.pending[event.SourceFQN], event)
	return event, nil
}

func (p *IngestionPipeline) PendingCount(sourceFQN string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	sourceFQN = strings.TrimSpace(sourceFQN)
	if sourceFQN != "" {
		return len(p.pending[sourceFQN])
	}
	total := 0
	for _, items := range p.pending {
		total += len(items)
	}
	return total
}

func (p *IngestionPipeline) Flush(ctx context.Context, cfg *config.Config, logger *zap.Logger, sourceFQN, baselineSnapshot string) (map[string]any, error) {
	sourceFQN = strings.TrimSpace(sourceFQN)
	baselineSnapshot = strings.TrimSpace(baselineSnapshot)
	if sourceFQN == "" {
		sourceFQN = strings.TrimSpace(cfg.Database.FQN)
	}
	if sourceFQN == "" {
		return nil, fmt.Errorf("source_fqn is required either in request or config")
	}

	pendingBefore := p.PendingCount(sourceFQN)
	if pendingBefore == 0 {
		return map[string]any{
			"source_fqn":     sourceFQN,
			"pending_events": 0,
			"message":        "no pending events to flush",
		}, nil
	}

	client := openmetadata.NewClient(cfg.OpenMetadata.BaseURL, cfg.OpenMetadata.JWTToken, logger)
	snap, err := snapshot.NewSnapshot(ctx, client, sourceFQN)
	if err != nil {
		return nil, fmt.Errorf("capture snapshot during flush: %w", err)
	}

	sourceDir := filepath.Join(cfg.Snapshot.Directory, "sources", fqnToPath(sourceFQN))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return nil, fmt.Errorf("create source snapshot directory: %w", err)
	}
	newSnapshotPath, err := snapshot.Save(snap, sourceDir)
	if err != nil {
		return nil, fmt.Errorf("save flushed snapshot: %w", err)
	}

	if baselineSnapshot == "" {
		baselineSnapshot = latestSnapshotExcluding(sourceDir, newSnapshotPath)
	}

	result := map[string]any{
		"source_fqn":        sourceFQN,
		"pending_events":    pendingBefore,
		"new_snapshot_path": newSnapshotPath,
	}

	if baselineSnapshot == "" {
		p.clearPending(sourceFQN)
		result["message"] = "snapshot captured, but no baseline snapshot available for compare"
		return result, nil
	}

	previous, err := snapshot.Load(baselineSnapshot)
	if err != nil {
		return nil, fmt.Errorf("load baseline snapshot: %w", err)
	}
	current, err := snapshot.Load(newSnapshotPath)
	if err != nil {
		return nil, fmt.Errorf("load new snapshot: %w", err)
	}

	diff := snapshot.CompareSnapshots(previous, current)
	analysis := impact.AnalyzeImpact(current, diff.Changes)

	report := map[string]any{
		"source_fqn":        sourceFQN,
		"baseline_snapshot": baselineSnapshot,
		"new_snapshot":      newSnapshotPath,
		"change_summary":    diff.Summary,
		"affected_tables":   diff.AffectedTables,
		"risk_level":        analysis.RiskLevel,
		"impacted_assets":   analysis.ImpactedAssets,
		"recommended":       analysis.RecommendedActions,
		"generated_at":      time.Now().UTC().Format(time.RFC3339),
	}

	reportPath := filepath.Join(p.baseDir, "reports", fmt.Sprintf("report_%d.json", time.Now().Unix()))
	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(reportPath, reportBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write report: %w", err)
	}

	p.clearPending(sourceFQN)

	result["baseline_snapshot"] = baselineSnapshot
	result["report_path"] = reportPath
	result["change_summary"] = diff.Summary
	result["risk_level"] = analysis.RiskLevel
	result["impacted_assets"] = len(analysis.ImpactedAssets)
	return result, nil
}

func (p *IngestionPipeline) clearPending(sourceFQN string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, sourceFQN)
}

func latestSnapshotExcluding(sourceDir, exclude string) string {
	pattern := filepath.Join(sourceDir, "snapshot_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i] == exclude {
			continue
		}
		return matches[i]
	}
	return ""
}
