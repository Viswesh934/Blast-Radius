package commands

import (
	"encoding/json"
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
	releaseBaselinePath            string
	releaseCurrentPath             string
	releaseSourceFQN               string
	releaseProfilePath             string
	releaseFailOn                  string
	releaseMaxTotal                int
	releaseMinScore                int
	releaseRequireOwner            bool
	releaseRequireTableDescription bool
	releaseMaxRisk                 string
	releaseMaxImpactedAssets       int
	releaseReportFile              string
	releaseMarkdownFile            string
)

var releaseCheckCmd = &cobra.Command{
	Use:   "release-check",
	Short: "Run release safety gate for metadata changes",
	Long:  "Release-check combines drift, impact, and contract readiness checks and exits non-zero when release policy is violated.",
	RunE: func(cmd *cobra.Command, args []string) error {
		baselinePath := strings.TrimSpace(releaseBaselinePath)
		if baselinePath == "" {
			return fmt.Errorf("--baseline is required")
		}
		baseline, err := snapshot.Load(baselinePath)
		if err != nil {
			return fmt.Errorf("load baseline snapshot: %w", err)
		}

		effectiveFailOn := releaseFailOn
		effectiveMaxTotal := releaseMaxTotal
		effectiveMinScore := releaseMinScore
		effectiveRequireOwner := releaseRequireOwner
		effectiveRequireTableDescription := releaseRequireTableDescription
		if strings.TrimSpace(releaseProfilePath) != "" {
			profile, err := loadGuardProfile(strings.TrimSpace(releaseProfilePath))
			if err != nil {
				return err
			}
			if strings.TrimSpace(profile.Drift.FailOn) != "" {
				effectiveFailOn = profile.Drift.FailOn
			}
			if profile.Drift.MaxTotal != nil {
				effectiveMaxTotal = *profile.Drift.MaxTotal
			}
			if profile.Contracts.MinScore != nil {
				effectiveMinScore = *profile.Contracts.MinScore
			}
			if profile.Contracts.RequireOwner != nil {
				effectiveRequireOwner = *profile.Contracts.RequireOwner
			}
			if profile.Contracts.RequireTableDescription != nil {
				effectiveRequireTableDescription = *profile.Contracts.RequireTableDescription
			}
		}

		if severityRank(effectiveFailOn) < 0 {
			return fmt.Errorf("invalid --fail-on value %q", effectiveFailOn)
		}
		if effectiveMinScore < 0 || effectiveMinScore > 100 {
			return fmt.Errorf("invalid --min-score value %d (must be 0..100)", effectiveMinScore)
		}
		if riskRank(releaseMaxRisk) < 0 {
			return fmt.Errorf("invalid --max-risk value %q (expected low|medium|high|critical)", releaseMaxRisk)
		}
		if effectiveMaxTotal < -1 {
			return fmt.Errorf("invalid --max-total value %d (must be >= -1)", effectiveMaxTotal)
		}
		if releaseMaxImpactedAssets < -1 {
			return fmt.Errorf("invalid --max-impacted-assets value %d (must be >= -1)", releaseMaxImpactedAssets)
		}

		current, currentLabel, err := loadCurrentForReleaseCheck(strings.TrimSpace(releaseCurrentPath), strings.TrimSpace(releaseSourceFQN))
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

		driftPass := true
		driftReasons := make([]string, 0, 2)
		if failRank > 0 && failingChanges > 0 {
			driftPass = false
			driftReasons = append(driftReasons, fmt.Sprintf("%d changes at or above severity=%s", failingChanges, strings.ToUpper(strings.TrimSpace(effectiveFailOn))))
		}
		if effectiveMaxTotal >= 0 && len(diff.Changes) > effectiveMaxTotal {
			driftPass = false
			driftReasons = append(driftReasons, fmt.Sprintf("total_changes=%d exceeds max_total=%d", len(diff.Changes), effectiveMaxTotal))
		}

		contractResults, contractsFailingTables, contractsOverallScore := evaluateContractReadiness(current, effectiveMinScore, effectiveRequireOwner, effectiveRequireTableDescription)
		contractsPass := contractsOverallScore >= effectiveMinScore && len(contractsFailingTables) == 0
		contractsReasons := make([]string, 0, 2)
		if contractsOverallScore < effectiveMinScore {
			contractsReasons = append(contractsReasons, fmt.Sprintf("overall_score=%d below min_score=%d", contractsOverallScore, effectiveMinScore))
		}
		if len(contractsFailingTables) > 0 {
			contractsReasons = append(contractsReasons, fmt.Sprintf("%d tables below min_score", len(contractsFailingTables)))
		}

		analysis := impact.AnalyzeImpact(current, diff.Changes)
		impactPass := riskRank(analysis.RiskLevel) <= riskRank(releaseMaxRisk)
		impactReasons := make([]string, 0, 2)
		if !impactPass {
			impactReasons = append(impactReasons, fmt.Sprintf("risk_level=%s exceeds max_risk=%s", strings.ToUpper(strings.TrimSpace(analysis.RiskLevel)), strings.ToUpper(strings.TrimSpace(releaseMaxRisk))))
		}
		if releaseMaxImpactedAssets >= 0 && len(analysis.ImpactedAssets) > releaseMaxImpactedAssets {
			impactPass = false
			impactReasons = append(impactReasons, fmt.Sprintf("impacted_assets=%d exceeds max_impacted_assets=%d", len(analysis.ImpactedAssets), releaseMaxImpactedAssets))
		}

		pass := driftPass && contractsPass && impactPass
		allReasons := make([]string, 0, len(driftReasons)+len(contractsReasons)+len(impactReasons))
		allReasons = append(allReasons, driftReasons...)
		allReasons = append(allReasons, contractsReasons...)
		allReasons = append(allReasons, impactReasons...)

		report := map[string]any{
			"gate":     "release-check",
			"pass":     pass,
			"baseline": baselinePath,
			"current":  currentLabel,
			"status":   guardStatus(pass),
			"reasons":  allReasons,
			"drift": map[string]any{
				"pass":            driftPass,
				"total_changes":   len(diff.Changes),
				"affected_tables": len(diff.AffectedTables),
				"summary":         diff.Summary,
				"severity_counts": severityCounts,
				"policy": map[string]any{
					"fail_on":   strings.ToUpper(strings.TrimSpace(effectiveFailOn)),
					"max_total": effectiveMaxTotal,
				},
				"reasons": driftReasons,
			},
			"contracts": map[string]any{
				"pass":           contractsPass,
				"overall_score":  contractsOverallScore,
				"min_score":      effectiveMinScore,
				"failing_tables": contractsFailingTables,
				"policy": map[string]any{
					"require_owner":             effectiveRequireOwner,
					"require_table_description": effectiveRequireTableDescription,
				},
				"reasons": contractsReasons,
				"results": contractResults,
			},
			"impact": map[string]any{
				"pass":            impactPass,
				"risk_level":      analysis.RiskLevel,
				"max_risk":        strings.ToUpper(strings.TrimSpace(releaseMaxRisk)),
				"impacted_assets": len(analysis.ImpactedAssets),
				"policy": map[string]any{
					"max_impacted_assets": releaseMaxImpactedAssets,
				},
				"recommended_actions": analysis.RecommendedActions,
				"reasons":             impactReasons,
			},
			"profile": strings.TrimSpace(releaseProfilePath),
		}

		if strings.TrimSpace(releaseReportFile) != "" {
			if err := writeJSONFile(strings.TrimSpace(releaseReportFile), report); err != nil {
				return err
			}
		}
		if strings.TrimSpace(releaseMarkdownFile) != "" {
			if err := writeTextFile(strings.TrimSpace(releaseMarkdownFile), buildReleaseCheckMarkdown(report)); err != nil {
				return err
			}
		}

		if wantsJSONOutput() {
			if err := printStructured(report); err != nil {
				return err
			}
		} else {
			fmt.Println("gate: release-check")
			fmt.Printf("baseline: %s\n", baselinePath)
			fmt.Printf("current: %s\n", currentLabel)
			fmt.Printf("status: %s\n", guardStatus(pass))
			fmt.Printf("drift: pass=%t total_changes=%d affected_tables=%d\n", driftPass, len(diff.Changes), len(diff.AffectedTables))
			fmt.Printf("contracts: pass=%t overall_score=%d failing_tables=%d\n", contractsPass, contractsOverallScore, len(contractsFailingTables))
			fmt.Printf("impact: pass=%t risk_level=%s impacted_assets=%d\n", impactPass, analysis.RiskLevel, len(analysis.ImpactedAssets))
			if len(allReasons) > 0 {
				for _, reason := range allReasons {
					fmt.Printf("reason: %s\n", reason)
				}
			}
			if strings.TrimSpace(releaseReportFile) != "" {
				fmt.Printf("report_file: %s\n", strings.TrimSpace(releaseReportFile))
			}
			if strings.TrimSpace(releaseMarkdownFile) != "" {
				fmt.Printf("markdown_file: %s\n", strings.TrimSpace(releaseMarkdownFile))
			}
		}

		if !pass {
			return fmt.Errorf("release check failed")
		}
		return nil
	},
}

func init() {
	releaseCheckCmd.Flags().StringVar(&releaseBaselinePath, "baseline", "", "baseline snapshot file (required)")
	releaseCheckCmd.Flags().StringVar(&releaseCurrentPath, "current", "", "current snapshot file")
	releaseCheckCmd.Flags().StringVar(&releaseSourceFQN, "source", "", "build current snapshot live from this OpenMetadata source FQN")
	releaseCheckCmd.Flags().StringVar(&releaseProfilePath, "profile", "", "guard profile yaml that sets drift/contracts policy")
	releaseCheckCmd.Flags().StringVar(&releaseFailOn, "fail-on", "critical", "minimum severity to fail on: none|info|warning|critical")
	releaseCheckCmd.Flags().IntVar(&releaseMaxTotal, "max-total", -1, "fail if total changes exceed this value (-1 disables)")
	releaseCheckCmd.Flags().IntVar(&releaseMinScore, "min-score", 80, "minimum contract readiness score required to pass (0-100)")
	releaseCheckCmd.Flags().BoolVar(&releaseRequireOwner, "require-owner", false, "require each table to have an owner")
	releaseCheckCmd.Flags().BoolVar(&releaseRequireTableDescription, "require-table-description", true, "require each table to have a non-empty description")
	releaseCheckCmd.Flags().StringVar(&releaseMaxRisk, "max-risk", "high", "maximum allowed impact risk: low|medium|high|critical")
	releaseCheckCmd.Flags().IntVar(&releaseMaxImpactedAssets, "max-impacted-assets", -1, "fail if impacted assets exceed this value (-1 disables)")
	releaseCheckCmd.Flags().StringVar(&releaseReportFile, "report-file", "", "optional path to write full JSON report")
	releaseCheckCmd.Flags().StringVar(&releaseMarkdownFile, "markdown-file", "", "optional path to write markdown summary for CI/PR comments")
}

func loadCurrentForReleaseCheck(currentPath, sourceFQN string) (*snapshot.StateSnapshot, string, error) {
	if currentPath != "" && sourceFQN != "" {
		return nil, "", fmt.Errorf("use either --current or --source, not both")
	}
	if currentPath != "" {
		s, err := snapshot.Load(currentPath)
		if err != nil {
			return nil, "", fmt.Errorf("load current snapshot: %w", err)
		}
		return s, currentPath, nil
	}
	if sourceFQN == "" {
		sourceFQN = strings.TrimSpace(cfg.Database.FQN)
	}
	if sourceFQN == "" {
		return nil, "", fmt.Errorf("provide --current or --source (or set BR_DATABASE_FQN)")
	}
	live, err := buildLiveSnapshot(strings.TrimSpace(sourceFQN))
	if err != nil {
		return nil, "", err
	}
	return live, "live:" + strings.TrimSpace(sourceFQN), nil
}

func evaluateContractReadiness(
	snap *snapshot.StateSnapshot,
	minScore int,
	requireOwner bool,
	requireTableDescription bool,
) ([]map[string]any, []string, int) {
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

		tChecks++
		if len(tbl.Columns) > 0 {
			tPass++
		} else {
			issues = append(issues, "no columns")
		}

		if requireTableDescription {
			tChecks++
			if strings.TrimSpace(tbl.Description) != "" {
				tPass++
			} else {
				issues = append(issues, "missing table description")
			}
		}

		if requireOwner {
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
		pass := tScore >= minScore
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
	return tableResults, failingTables, overallScore
}

func riskRank(level string) int {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "LOW":
		return 0
	case "MEDIUM":
		return 1
	case "HIGH":
		return 2
	case "CRITICAL":
		return 3
	default:
		return -1
	}
}

func writeJSONFile(path string, payload any) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := jsonMarshalIndent(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report file: %w", err)
	}
	return nil
}

func writeTextFile(path, content string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create markdown directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write markdown file: %w", err)
	}
	return nil
}

func buildReleaseCheckMarkdown(report map[string]any) string {
	status := fmt.Sprintf("%v", report["status"])
	baseline := fmt.Sprintf("%v", report["baseline"])
	current := fmt.Sprintf("%v", report["current"])

	drift := report["drift"].(map[string]any)
	contracts := report["contracts"].(map[string]any)
	impactSection := report["impact"].(map[string]any)

	lines := []string{
		"## Blast Radius Release Check",
		"",
		fmt.Sprintf("- Status: **%s**", status),
		fmt.Sprintf("- Baseline: `%s`", baseline),
		fmt.Sprintf("- Current: `%s`", current),
		"",
		"### Drift",
		fmt.Sprintf("- Pass: `%v`", drift["pass"]),
		fmt.Sprintf("- Total changes: `%v`", drift["total_changes"]),
		fmt.Sprintf("- Affected tables: `%v`", drift["affected_tables"]),
		"",
		"### Contracts",
		fmt.Sprintf("- Pass: `%v`", contracts["pass"]),
		fmt.Sprintf("- Overall score: `%v`", contracts["overall_score"]),
		fmt.Sprintf("- Failing tables: `%d`", len(contracts["failing_tables"].([]string))),
		"",
		"### Impact",
		fmt.Sprintf("- Pass: `%v`", impactSection["pass"]),
		fmt.Sprintf("- Risk level: `%v`", impactSection["risk_level"]),
		fmt.Sprintf("- Impacted assets: `%v`", impactSection["impacted_assets"]),
	}

	if reasons, ok := report["reasons"].([]string); ok && len(reasons) > 0 {
		lines = append(lines, "", "### Reasons")
		for _, reason := range reasons {
			lines = append(lines, "- "+reason)
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func jsonMarshalIndent(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal report json: %w", err)
	}
	return data, nil
}
