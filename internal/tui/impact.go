package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func NewImpactModel(result *ImpactResult) ImpactModel {
	return ImpactModel{result: result}
}

func (m ImpactModel) Init() tea.Cmd { return nil }

func (m ImpactModel) Update(msg tea.Msg) (ImpactModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.scrollPos > 0 {
				m.scrollPos--
			}
		case "down", "j":
			if m.scrollPos < max(0, len(m.result.Changes)-1) {
				m.scrollPos++
			}
		case "left", "h":
			if m.selectedAsset > 0 {
				m.selectedAsset--
			}
		case "right", "l":
			if m.selectedAsset < max(0, len(m.result.ImpactedAssets)-1) {
				m.selectedAsset++
			}
		case "esc":
			return m, func() tea.Msg { return BackToHomeMsg{} }
		}
	}
	return m, nil
}

func (m ImpactModel) View() string {
	if m.result == nil {
		return NewAlert(AlertInfo, "No impact result", "Run comparison from snapshots first.").Render()
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		titleStyle.Render("IMPACT ANALYSIS"),
		"  ",
		m.riskBadgeStyle(),
		"  ",
		subtitleStyle.Render("time="+m.result.ExecutionTime.String()),
	)

	changes := m.renderChangesSummary()
	assets := m.renderImpactedAssets()
	recs := m.renderRecommendations()
	help := helpStyle.Render("up/down scroll changes | left/right choose asset | esc back")

	return lipgloss.JoinVertical(lipgloss.Left, header, "", changes, "", assets, "", recs, "", help)
}

func (m ImpactModel) riskBadgeStyle() string {
	return RiskBadge(m.result.RiskLevel)
}

func (m ImpactModel) renderChangesSummary() string {
	if len(m.result.Changes) == 0 {
		return panelStyle.Render("No schema changes detected")
	}
	var rows []string
	for i, change := range m.result.Changes {
		if i < m.scrollPos {
			continue
		}
		if len(rows) > 14 {
			break
		}
		icon := "*"
		style := changeModifiedStyle
		switch strings.ToUpper(change.Type) {
		case "DELETED":
			icon = "-"
			style = changeDeletedStyle
		case "ADDED":
			icon = "+"
			style = changeAddedStyle
		}
		line := fmt.Sprintf("%s [%s] %s", icon, change.Type, change.Entity)
		if change.Field != "" {
			line += "." + change.Field
		}
		if change.OldValue != nil || change.NewValue != nil {
			line += fmt.Sprintf(" (%v -> %v)", change.OldValue, change.NewValue)
		}
		rows = append(rows, style.Render(line))
	}

	return panelStyle.Render(titleStyle.Render("Changes") + "\n" + strings.Join(rows, "\n"))
}

func (m ImpactModel) renderImpactedAssets() string {
	if len(m.result.ImpactedAssets) == 0 {
		return panelStyle.Render("No impacted assets found")
	}
	table := NewTable([]TableColumn{{Header: "Asset", Width: 42}, {Header: "Type", Width: 10}, {Header: "Status", Width: 12}})
	for i, asset := range m.result.ImpactedAssets {
		prefix := ""
		if i == m.selectedAsset {
			prefix = ">"
		}
		status := "affected"
		if asset.DirectlyAffected {
			status = "direct"
		}
		table.AddRow([]string{prefix + TruncateString(asset.FQN, 40), asset.Type, status})
	}
	return panelStyle.Render(titleStyle.Render("Impacted Assets") + "\n" + table.Render())
}

func (m ImpactModel) renderRecommendations() string {
	if len(m.result.Recommendations) == 0 {
		return panelStyle.Render("No recommendations")
	}
	var rows []string
	for i, rec := range m.result.Recommendations {
		rows = append(rows, fmt.Sprintf("%d. %s", i+1, rec))
	}
	return panelStyle.Render(titleStyle.Render("Recommendations") + "\n" + strings.Join(rows, "\n"))
}
