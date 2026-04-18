package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type homeModel struct {
	selectedIdx int
	menuItems   []MenuItem
	model       *Model
}

func newHomeModel(model *Model) homeModel {
	h := homeModel{model: model}
	h.menuItems = []MenuItem{
		{
			Title:       "Take New Snapshot",
			Description: "Create a local snapshot placeholder for this environment",
			Icon:        "SNAP",
			Action: func(m *Model) tea.Cmd {
				m.currentScreen = LoadingScreen
				m.loading = true
				m.message = "Creating snapshot..."
				m.messageType = MessageInfo
				return captureSnapshotCmd(m.cfg.Snapshot.Directory)
			},
		},
		{
			Title:       "Compare Two Snapshots",
			Description: "Pick two snapshots and run impact analysis",
			Icon:        "DIFF",
			Action: func(m *Model) tea.Cmd {
				return tea.Sequence(
					func() tea.Msg { return StartCompareModeMsg{} },
					loadSnapshotsCmd(m.cfg.Snapshot.Directory),
				)
			},
		},
		{
			Title:       "View All Snapshots",
			Description: "Browse available snapshots",
			Icon:        "LIST",
			Action: func(m *Model) tea.Cmd {
				return tea.Sequence(
					func() tea.Msg { return StartBrowseModeMsg{} },
					loadSnapshotsCmd(m.cfg.Snapshot.Directory),
				)
			},
		},
		{
			Title:       "Toggle Help",
			Description: "Show keyboard shortcuts",
			Icon:        "HELP",
			Action: func(m *Model) tea.Cmd {
				m.helpVisible = !m.helpVisible
				return nil
			},
		},
	}
	return h
}

func (m homeModel) Init() tea.Cmd { return nil }

func (m homeModel) Update(msg tea.Msg) (homeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "down", "j":
			if m.selectedIdx < len(m.menuItems)-1 {
				m.selectedIdx++
			}
		case "enter":
			return m, m.menuItems[m.selectedIdx].Action(m.model)
		}
	}
	return m, nil
}

func (m homeModel) View() string {
	title := titleStyle.Render("BLAST RADIUS")
	subtitle := subtitleStyle.Render("Data impact analysis in terminal")

	statsRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		NewStatBox("Snapshots", fmtInt(len(m.model.snapshots)), "SNAP", Colors.Info).Render(20),
		NewStatBox("Last Check", m.lastCheckTime(), "TIME", Colors.Secondary).Render(20),
		NewStatBox("At Risk", m.atRiskCount(), "RISK", Colors.Warning).Render(20),
	)

	var menuRows []string
	for i, item := range m.menuItems {
		style := menuItemStyle
		prefix := "  "
		if i == m.selectedIdx {
			style = selectedMenuItemStyle
			prefix = "> "
		}
		button := NewButton(item.Icon + "  " + item.Title)
		button.IsSelected = i == m.selectedIdx
		menuRows = append(menuRows, style.Render(prefix+button.Render()))
		if i == m.selectedIdx {
			menuRows = append(menuRows, subtitleStyle.Render("    "+item.Description))
		}
	}

	help := helpStyle.Render("up/down or j/k to navigate | enter select | q quit | ? help")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", statsRow, "", strings.Join(menuRows, "\n"), "", help)
}
func (m homeModel) lastCheckTime() string {
	if len(m.model.snapshots) == 0 {
		return "Never"
	}
	return m.model.snapshots[0].Timestamp.Format("15:04")
}

func (m homeModel) atRiskCount() string {
	count := 0
	for _, snap := range m.model.snapshots {
		if strings.Contains(strings.ToLower(snap.Source), "critical") {
			count++
		}
	}
	return fmtInt(count)
}
