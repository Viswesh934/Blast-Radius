package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	demoOlderSnapshot string
	demoNewerSnapshot string
	demoSourceFilter  string
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run a local Blast workflow demo",
	Long:  "Shows how Blast compares two snapshots and computes downstream impact without needing live OpenMetadata access.",
	RunE: func(cmd *cobra.Command, args []string) error {
		older, newer, err := resolveDemoSnapshots(demoOlderSnapshot, demoNewerSnapshot, demoSourceFilter)
		if err != nil {
			return err
		}

		previous, err := snapshot.Load(older)
		if err != nil {
			return fmt.Errorf("load older snapshot: %w", err)
		}
		current, err := snapshot.Load(newer)
		if err != nil {
			return fmt.Errorf("load newer snapshot: %w", err)
		}

		diff := snapshot.CompareSnapshots(previous, current)
		analysis := impact.AnalyzeImpact(current, diff.Changes)

		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"demo": map[string]any{
					"older_snapshot": older,
					"newer_snapshot": newer,
					"summary":        diff.Summary,
					"impact":         analysis,
				},
			})
		}

		fmt.Println("Blast demo (snapshot compare + impact)")
		fmt.Printf("older_snapshot: %s\n", older)
		fmt.Printf("newer_snapshot: %s\n", newer)
		fmt.Println("change_summary:")
		fmt.Printf("  - added: %d\n", diff.Summary[string(snapshot.ChangeTypeAdded)])
		fmt.Printf("  - deleted: %d\n", diff.Summary[string(snapshot.ChangeTypeDeleted)])
		fmt.Printf("  - modified: %d\n", diff.Summary[string(snapshot.ChangeTypeModified)])
		fmt.Printf("  - affected_tables: %d\n", len(diff.AffectedTables))
		fmt.Println("impact:")
		fmt.Printf("  - risk_level: %s\n", analysis.RiskLevel)
		fmt.Printf("  - impacted_assets: %d\n", len(analysis.ImpactedAssets))
		if len(analysis.RecommendedActions) > 0 {
			fmt.Println("  - recommended_actions:")
			for _, action := range analysis.RecommendedActions {
				fmt.Printf("    - %s\n", action)
			}
		}

		return nil
	},
}

func init() {
	demoCmd.Flags().StringVar(&demoOlderSnapshot, "older", "", "path to older snapshot file")
	demoCmd.Flags().StringVar(&demoNewerSnapshot, "newer", "", "path to newer snapshot file")
	demoCmd.Flags().StringVar(&demoSourceFilter, "source", "", "source filter under snapshots/sources (for example my_service.analytics.public)")
}

func resolveDemoSnapshots(olderInput, newerInput, sourceFilter string) (string, string, error) {
	older := strings.TrimSpace(olderInput)
	newer := strings.TrimSpace(newerInput)

	if older != "" || newer != "" {
		if older == "" || newer == "" {
			return "", "", fmt.Errorf("both --older and --newer are required when one is provided")
		}
		if _, err := os.Stat(older); err != nil {
			return "", "", fmt.Errorf("older snapshot not found: %s", older)
		}
		if _, err := os.Stat(newer); err != nil {
			return "", "", fmt.Errorf("newer snapshot not found: %s", newer)
		}
		return older, newer, nil
	}

	baseDir := cfg.Snapshot.Directory
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "./snapshots"
	}

	searchRoot := filepath.Join(baseDir, "sources")
	if strings.TrimSpace(sourceFilter) != "" {
		parts := strings.Split(strings.TrimSpace(sourceFilter), ".")
		searchRoot = filepath.Join(append([]string{searchRoot}, parts...)...)
	}

	candidates := make([]snapshotCandidate, 0)
	err := filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(d.Name(), "snapshot_") || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		candidates = append(candidates, snapshotCandidate{Path: path, ModTimeUnix: info.ModTime().UnixNano()})
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("scan snapshots: %w", err)
	}
	if len(candidates) < 2 {
		return "", "", fmt.Errorf("need at least 2 snapshots under %s to run demo", searchRoot)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ModTimeUnix < candidates[j].ModTimeUnix
	})

	older = candidates[len(candidates)-2].Path
	newer = candidates[len(candidates)-1].Path
	return older, newer, nil
}

type snapshotCandidate struct {
	Path        string
	ModTimeUnix int64
}
