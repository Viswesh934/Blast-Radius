package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/Viswesh934/blast-radius/internal/openmetadata"
)

type OpenMetadataAPI interface {
	GetTablesByDatabase(ctx context.Context, databaseFQN string) ([]openmetadata.Table, error)
	GetLineage(ctx context.Context, entityFQN string, upstreamDepth, downstreamDepth int) (*openmetadata.LineageData, error)
}

type StateSnapshot struct {
	ID        string                   `json:"id"`
	Timestamp time.Time                `json:"timestamp"`
	Source    string                   `json:"source"`
	Tables    map[string]*TableState   `json:"tables"`
	Lineage   map[string]*LineageState `json:"lineage"`
	Metadata  map[string]string        `json:"metadata"`
}

type TableState struct {
	FQN         string                  `json:"fqn"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Owner       string                  `json:"owner"`
	Columns     map[string]*ColumnState `json:"columns"`
	Updated     time.Time               `json:"updated"`
	Hash        string                  `json:"hash"`
}

type ColumnState struct {
	Name        string `json:"name"`
	DataType    string `json:"data_type"`
	Description string `json:"description"`
	Nullable    bool   `json:"nullable"`
	Constraints string `json:"constraints"`
	Hash        string `json:"hash"`
}

type LineageState struct {
	EntityFQN      string   `json:"entity_fqn"`
	UpstreamFQNs   []string `json:"upstream_fqns"`
	DownstreamFQNs []string `json:"downstream_fqns"`
	Hash           string   `json:"hash"`
}

func NewSnapshot(ctx context.Context, client OpenMetadataAPI, databaseFQN string) (*StateSnapshot, error) {
	snap := &StateSnapshot{
		ID:        fmt.Sprintf("snap_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		Source:    databaseFQN,
		Tables:    make(map[string]*TableState),
		Lineage:   make(map[string]*LineageState),
		Metadata:  map[string]string{"version": "0.1.0"},
	}

	tables, err := client.GetTablesByDatabase(ctx, databaseFQN)
	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		tableState := &TableState{
			FQN:         table.FQN,
			Name:        table.Name,
			Description: table.Description,
			Owner:       table.Owner,
			Columns:     make(map[string]*ColumnState),
			Updated:     toTime(table.UpdatedAt),
		}

		for _, column := range table.Columns {
			columnState := &ColumnState{
				Name:        column.Name,
				DataType:    column.DataType,
				Description: column.Description,
				Nullable:    column.Nullable,
				Constraints: column.Constraint,
			}
			columnState.Hash = hashColumnState(columnState)
			tableState.Columns[column.Name] = columnState
		}
		tableState.Hash = hashTableState(tableState)
		snap.Tables[table.FQN] = tableState

		lineage, err := client.GetLineage(ctx, table.FQN, 10, 10)
		if err != nil || lineage == nil {
			continue
		}
		lineageState := &LineageState{
			EntityFQN:      table.FQN,
			UpstreamFQNs:   uniqueSorted(lineage.UpstreamFQNs),
			DownstreamFQNs: uniqueSorted(lineage.DownstreamFQNs),
		}
		lineageState.Hash = hashLineageState(lineageState)
		snap.Lineage[table.FQN] = lineageState
	}

	return snap, nil
}

func toTime(unixMillis int64) time.Time {
	if unixMillis == 0 {
		return time.Time{}
	}
	return time.UnixMilli(unixMillis).UTC()
}

func hashTableState(ts *TableState) string {
	keys := make([]string, 0, len(ts.Columns))
	for key := range ts.Columns {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, key := range keys {
		col := ts.Columns[key]
		_, _ = h.Write([]byte(col.Hash))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func hashColumnState(cs *ColumnState) string {
	h := sha256.New()
	_, _ = h.Write([]byte(cs.Name))
	_, _ = h.Write([]byte(cs.DataType))
	_, _ = h.Write([]byte(cs.Description))
	if cs.Nullable {
		_, _ = h.Write([]byte("nullable"))
	}
	_, _ = h.Write([]byte(cs.Constraints))
	return hex.EncodeToString(h.Sum(nil))
}

func hashLineageState(ls *LineageState) string {
	h := sha256.New()
	for _, fqn := range ls.UpstreamFQNs {
		_, _ = h.Write([]byte("up:" + fqn))
	}
	for _, fqn := range ls.DownstreamFQNs {
		_, _ = h.Write([]byte("down:" + fqn))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		set[v] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for v := range set {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}
