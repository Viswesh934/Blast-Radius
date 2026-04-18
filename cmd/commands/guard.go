package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	driftFailOn   string
	driftMaxTotal int
	driftSource   string
	driftProfile  string

	contractsSource        string
	contractsMinScore      int
	contractsRequireOwner  bool
	contractsRequireTableD bool
	contractsProfile       string
)

var guardCmd = &cobra.Command{
	Use:   "guard",
	Short: "Run CI-style metadata guards",
	Long:  "Guard provides metadata drift checks and contract readiness scoring for snapshots or live OpenMetadata state.",
}

var guardDriftCmd = &cobra.Command{
	Use:   "drift <baseline-snapshot> [current-snapshot]",
	Short: "Detect metadata drift and optionally fail by policy",
	Long:  "Compare baseline and current snapshots; current can come from file or live OpenMetadata via --source.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		effectiveFailOn := driftFailOn
		effectiveMaxTotal := driftMaxTotal
		if strings.TrimSpace(driftProfile) != "" {
			profile, err := loadGuardProfile(strings.TrimSpace(driftProfile))
			if err != nil {
				return err
			}
			if strings.TrimSpace(profile.Drift.FailOn) != "" {
				effectiveFailOn = profile.Drift.FailOn
			}
			if profile.Drift.MaxTotal != nil {
				effectiveMaxTotal = *profile.Drift.MaxTotal
			}
		}
		if severityRank(effectiveFailOn) < 0 {
			return fmt.Errorf("invalid --fail-on value %q", effectiveFailOn)
		}

		baselinePath := strings.TrimSpace(args[0])
		baseline, err := snapshot.Load(baselinePath)
		if err != nil {
			return fmt.Errorf("load baseline snapshot: %w", err)
		}

		current, currentLabel, err := loadCurrentForDrift(args)
		if err != nil {
			return err
		}

		diff := snapshot.CompareSnapshots(baseline, current)
		severityCounts := map[string]int{"INFO": 0, "WARNING": 0, "CRITICAL": 0}
		for _, ch := range diff.Changes {
			sev := strings.ToUpper(strings.TrimSpace(ch.Severity))
			if sev == "" {
				sev = "INFO"
			}
			severityCounts[sev]++
		}

		failingChanges := 0
		failRank := severityRank(effectiveFailOn)
		for _, ch := range diff.Changes {
			if changeSeverityRank(ch.Severity) >= failRank {
				failingChanges++
			}
		}

		pass := true
		failureReasons := make([]string, 0, 2)
		if failRank > 0 && failingChanges > 0 {
			pass = false
			failureReasons = append(failureReasons, fmt.Sprintf("%d changes at or above severity=%s", failingChanges, strings.ToUpper(strings.TrimSpace(effectiveFailOn))))
		}
		if effectiveMaxTotal >= 0 && len(diff.Changes) > effectiveMaxTotal {
			pass = false
			failureReasons = append(failureReasons, fmt.Sprintf("total_changes=%d exceeds max_total=%d", len(diff.Changes), effectiveMaxTotal))
		}

		payload := map[string]any{
			"guard":           "drift",
			"pass":            pass,
			"baseline":        baselinePath,
			"current":         currentLabel,
			"total_changes":   len(diff.Changes),
			"affected_tables": len(diff.AffectedTables),
			"severity_counts": severityCounts,
			"policy": map[string]any{
				"fail_on":   strings.ToUpper(strings.TrimSpace(effectiveFailOn)),
				"max_total": effectiveMaxTotal,
				"profile":   strings.TrimSpace(driftProfile),
			},
			"summary": diff.Summary,
			"reasons": failureReasons,
			"diff":    diff,
		}
		if wantsJSONOutput() {
			if err := printStructured(payload); err != nil {
				return err
			}
		} else {
			fmt.Println("guard: drift")
			fmt.Printf("baseline: %s\n", baselinePath)
			fmt.Printf("current: %s\n", currentLabel)
			fmt.Printf("total_changes: %d\n", len(diff.Changes))
			fmt.Printf("affected_tables: %d\n", len(diff.AffectedTables))
			fmt.Printf("severity: critical=%d warning=%d info=%d\n", severityCounts["CRITICAL"], severityCounts["WARNING"], severityCounts["INFO"])
			fmt.Printf("policy: fail_on=%s max_total=%d profile=%s\n", strings.ToUpper(strings.TrimSpace(effectiveFailOn)), effectiveMaxTotal, strings.TrimSpace(driftProfile))
			fmt.Printf("status: %s\n", guardStatus(pass))
			for _, reason := range failureReasons {
				fmt.Printf("reason: %s\n", reason)
			}
		}

		if !pass {
			return fmt.Errorf("drift guard failed")
		}
		return nil
	},
}

var guardContractsCmd = &cobra.Command{
	Use:   "contracts [snapshot-file]",
	Short: "Score contract readiness for metadata tables",
	Long:  "Evaluate table and column metadata completeness from snapshot file or live OpenMetadata source.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		effectiveMinScore := contractsMinScore
		effectiveRequireOwner := contractsRequireOwner
		effectiveRequireTableD := contractsRequireTableD
		if strings.TrimSpace(contractsProfile) != "" {
			profile, err := loadGuardProfile(strings.TrimSpace(contractsProfile))
			if err != nil {
				return err
			}
			if profile.Contracts.MinScore != nil {
				effectiveMinScore = *profile.Contracts.MinScore
			}
			if profile.Contracts.RequireOwner != nil {
				effectiveRequireOwner = *profile.Contracts.RequireOwner
			}
			if profile.Contracts.RequireTableDescription != nil {
				effectiveRequireTableD = *profile.Contracts.RequireTableDescription
			}
		}
		if effectiveMinScore < 0 || effectiveMinScore > 100 {
			return fmt.Errorf("invalid min score %d (must be 0..100)", effectiveMinScore)
		}

		snap, label, err := loadSnapshotForContracts(args)
		if err != nil {
			return err
		}

		tableFQNs := make([]string, 0, len(snap.Tables))
		for fqn := range snap.Tables {
			tableFQNs = append(tableFQNs, fqn)
		}
		sort.Strings(tableFQNs)

		tableResults := make([]map[string]any, 0, len(tableFQNs))
		failingTables := make([]string, 0)
		totalChecks := 0
		passedChecks := 0

		for _, fqn := range tableFQNs {
			tbl := snap.Tables[fqn]
			issues := make([]string, 0)
			tChecks := 0
			tPass := 0

			// Rule: table has at least one column.
			tChecks++
			if len(tbl.Columns) > 0 {
				tPass++
			} else {
				issues = append(issues, "no columns")
			}

			if effectiveRequireTableD {
				tChecks++
				if strings.TrimSpace(tbl.Description) != "" {
					tPass++
				} else {
					issues = append(issues, "missing table description")
				}
			}

			if effectiveRequireOwner {
				tChecks++
				if strings.TrimSpace(tbl.Owner) != "" {
					tPass++
				} else {
					issues = append(issues, "missing owner")
				}
			}

			missingColumnDescriptions := 0
			missingColumnTypes := 0
			for _, col := range tbl.Columns {
				tChecks++
				if strings.TrimSpace(col.DataType) != "" {
					tPass++
				} else {
					missingColumnTypes++
				}

				tChecks++
				if strings.TrimSpace(col.Description) != "" {
					tPass++
				} else {
					missingColumnDescriptions++
				}
			}
			if missingColumnTypes > 0 {
				issues = append(issues, fmt.Sprintf("%d columns missing data type", missingColumnTypes))
			}
			if missingColumnDescriptions > 0 {
				issues = append(issues, fmt.Sprintf("%d columns missing description", missingColumnDescriptions))
			}

			tScore := 0
			if tChecks > 0 {
				tScore = int((100.0 * float64(tPass)) / float64(tChecks))
			}
			pass := tScore >= effectiveMinScore
			if !pass {
				failingTables = append(failingTables, fqn)
			}

			totalChecks += tChecks
			passedChecks += tPass
			tableResults = append(tableResults, map[string]any{
				"table":   fqn,
				"score":   tScore,
				"pass":    pass,
				"checks":  tChecks,
				"passed":  tPass,
				"issues":  issues,
				"columns": len(tbl.Columns),
			})
		}

		overallScore := 0
		if totalChecks > 0 {
			overallScore = int((100.0 * float64(passedChecks)) / float64(totalChecks))
		}
		pass := overallScore >= effectiveMinScore && len(failingTables) == 0

		payload := map[string]any{
			"guard":          "contracts",
			"pass":           pass,
			"source":         label,
			"overall_score":  overallScore,
			"min_score":      effectiveMinScore,
			"table_count":    len(tableResults),
			"failing_tables": failingTables,
			"policy": map[string]any{
				"require_owner":             effectiveRequireOwner,
				"require_table_description": effectiveRequireTableD,
				"profile":                   strings.TrimSpace(contractsProfile),
			},
			"results": tableResults,
		}
		if wantsJSONOutput() {
			if err := printStructured(payload); err != nil {
				return err
			}
		} else {
			fmt.Println("guard: contracts")
			fmt.Printf("source: %s\n", label)
			fmt.Printf("overall_score: %d\n", overallScore)
			fmt.Printf("min_score: %d\n", effectiveMinScore)
			fmt.Printf("policy: require_owner=%t require_table_description=%t profile=%s\n", effectiveRequireOwner, effectiveRequireTableD, strings.TrimSpace(contractsProfile))
			fmt.Printf("tables: %d\n", len(tableResults))
			fmt.Printf("failing_tables: %d\n", len(failingTables))
			fmt.Printf("status: %s\n", guardStatus(pass))
			for _, fqn := range failingTables {
				fmt.Printf("- %s\n", fqn)
			}
		}

		if !pass {
			return fmt.Errorf("contract readiness guard failed")
		}
		return nil
	},
}

func init() {
	guardDriftCmd.Flags().StringVar(&driftFailOn, "fail-on", "critical", "minimum severity to fail on: none|info|warning|critical")
	guardDriftCmd.Flags().IntVar(&driftMaxTotal, "max-total", -1, "fail if total changes exceed this value (-1 disables)")
	guardDriftCmd.Flags().StringVar(&driftSource, "source", "", "compare baseline with live snapshot from this OpenMetadata database/schema FQN")
	guardDriftCmd.Flags().StringVar(&driftProfile, "profile", "", "guard profile yaml for drift policy")

	guardContractsCmd.Flags().StringVar(&contractsSource, "source", "", "run contract check on live OpenMetadata source FQN instead of snapshot file")
	guardContractsCmd.Flags().IntVar(&contractsMinScore, "min-score", 80, "minimum readiness score required to pass (0-100)")
	guardContractsCmd.Flags().BoolVar(&contractsRequireOwner, "require-owner", false, "require each table to have an owner")
	guardContractsCmd.Flags().BoolVar(&contractsRequireTableD, "require-table-description", true, "require each table to have a non-empty description")
	guardContractsCmd.Flags().StringVar(&contractsProfile, "profile", "", "guard profile yaml for contracts policy")

	guardCmd.AddCommand(guardDriftCmd)
	guardCmd.AddCommand(guardContractsCmd)
	guardCmd.AddCommand(guardProfileCmd)
}

func loadCurrentForDrift(args []string) (*snapshot.StateSnapshot, string, error) {
	if len(args) == 2 {
		if strings.TrimSpace(driftSource) != "" {
			return nil, "", fmt.Errorf("use either current snapshot path or --source, not both")
		}
		p := strings.TrimSpace(args[1])
		s, err := snapshot.Load(p)
		if err != nil {
			return nil, "", fmt.Errorf("load current snapshot: %w", err)
		}
		return s, p, nil
	}

	if strings.TrimSpace(driftSource) == "" {
		return nil, "", fmt.Errorf("provide current snapshot path or --source")
	}
	live, err := buildLiveSnapshot(strings.TrimSpace(driftSource))
	if err != nil {
		return nil, "", err
	}
	return live, "live:" + strings.TrimSpace(driftSource), nil
}

func loadSnapshotForContracts(args []string) (*snapshot.StateSnapshot, string, error) {
	if len(args) == 1 {
		if strings.TrimSpace(contractsSource) != "" {
			return nil, "", fmt.Errorf("use either snapshot file or --source, not both")
		}
		p := strings.TrimSpace(args[0])
		s, err := snapshot.Load(p)
		if err != nil {
			return nil, "", fmt.Errorf("load snapshot: %w", err)
		}
		return s, p, nil
	}

	if strings.TrimSpace(contractsSource) != "" {
		s, err := buildLiveSnapshot(strings.TrimSpace(contractsSource))
		if err != nil {
			return nil, "", err
		}
		return s, "live:" + strings.TrimSpace(contractsSource), nil
	}

	latest, err := findLatestSnapshot(cfg.Snapshot.Directory)
	if err != nil {
		return nil, "", err
	}
	s, err := snapshot.Load(latest)
	if err != nil {
		return nil, "", fmt.Errorf("load latest snapshot: %w", err)
	}
	return s, latest, nil
}

func findLatestSnapshot(root string) (string, error) {
	type candidate struct {
		path string
		mod  int64
	}
	var picks []candidate
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		if !strings.Contains(strings.ToLower(d.Name()), "snapshot_") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		picks = append(picks, candidate{path: path, mod: info.ModTime().UnixNano()})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("scan snapshots: %w", err)
	}
	if len(picks) == 0 {
		return "", fmt.Errorf("no snapshots found under %s", root)
	}
	sort.Slice(picks, func(i, j int) bool { return picks[i].mod > picks[j].mod })
	return picks[0].path, nil
}

func buildLiveSnapshot(sourceFQN string) (*snapshot.StateSnapshot, error) {
	client, err := newOMClient()
	if err != nil {
		return nil, err
	}
	s, err := snapshot.NewSnapshot(context.Background(), client, sourceFQN)
	if err != nil {
		return nil, fmt.Errorf("build live snapshot: %w", err)
	}
	return s, nil
}

func severityRank(severity string) int {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "CRITICAL":
		return 3
	case "WARNING":
		return 2
	case "INFO":
		return 1
	case "NONE":
		return 0
	default:
		return -1
	}
}

func changeSeverityRank(severity string) int {
	rank := severityRank(severity)
	if rank < 0 {
		return 1
	}
	return rank
}

func guardStatus(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}
