package impact

import "github.com/Viswesh934/blast-radius/internal/snapshot"

type ImpactAnalysis struct {
	Changes            []snapshot.Change `json:"changes"`
	ImpactedAssets     []ImpactedAsset   `json:"impacted_assets"`
	RiskLevel          string            `json:"risk_level"`
	RecommendedActions []string          `json:"recommended_actions"`
}

type ImpactedAsset struct {
	FQN              string `json:"fqn"`
	Type             string `json:"type"`
	Reason           string `json:"reason"`
	DirectlyAffected bool   `json:"directly_affected"`
}

func AnalyzeImpact(current *snapshot.StateSnapshot, changes []snapshot.Change) *ImpactAnalysis {
	analysis := &ImpactAnalysis{
		Changes:            changes,
		ImpactedAssets:     make([]ImpactedAsset, 0),
		RecommendedActions: make([]string, 0),
	}

	impacted := make(map[string]ImpactedAsset)
	criticalCount := 0

	for _, change := range changes {
		if change.Severity == "CRITICAL" {
			criticalCount++
		}
		asset := ImpactedAsset{
			FQN:              change.Entity,
			Type:             "table",
			Reason:           "Changed directly",
			DirectlyAffected: true,
		}
		impacted[asset.FQN] = asset

		if lineage, ok := current.Lineage[change.Entity]; ok {
			for _, downstream := range lineage.DownstreamFQNs {
				if _, exists := impacted[downstream]; exists {
					continue
				}
				impacted[downstream] = ImpactedAsset{
					FQN:              downstream,
					Type:             "asset",
					Reason:           "Depends on " + change.Entity,
					DirectlyAffected: false,
				}
			}
		}
	}

	for _, asset := range impacted {
		analysis.ImpactedAssets = append(analysis.ImpactedAssets, asset)
	}

	switch {
	case criticalCount > 0 || len(analysis.ImpactedAssets) > 10:
		analysis.RiskLevel = "CRITICAL"
	case len(analysis.ImpactedAssets) > 5:
		analysis.RiskLevel = "HIGH"
	case len(analysis.ImpactedAssets) > 0:
		analysis.RiskLevel = "MEDIUM"
	default:
		analysis.RiskLevel = "LOW"
	}

	analysis.RecommendedActions = recommendationsFor(analysis.RiskLevel)
	return analysis
}

func recommendationsFor(riskLevel string) []string {
	switch riskLevel {
	case "CRITICAL":
		return []string{
			"Review impacted downstream assets before deployment",
			"Notify owning teams for all critical entities",
			"Run full regression checks for pipelines and dashboards",
		}
	case "HIGH":
		return []string{
			"Schedule an impact review with downstream owners",
			"Validate contract expectations for changed tables",
		}
	case "MEDIUM":
		return []string{
			"Run targeted regression tests for impacted assets",
		}
	default:
		return []string{"No action required"}
	}
}
