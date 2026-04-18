package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func NewSnapshotListModel(snapshots []SnapshotInfo) SnapshotListModel {
	items := make([]list.Item, 0, len(snapshots))
	for _, snap := range snapshots {
		items = append(items, snapshotItem{snapshot: snap})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	l := list.New(items, delegate, 80, 18)
	l.Title = "Snapshots"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return SnapshotListModel{
		list:      l,
		snapshots: snapshots,
		mode:      ModeBrowse,
	}
}

func (m SnapshotListModel) Init() tea.Cmd { return nil }

func (m SnapshotListModel) Update(msg tea.Msg) (SnapshotListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(max(40, msg.Width-8))
		m.list.SetHeight(max(10, msg.Height-14))
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected, ok := m.list.SelectedItem().(snapshotItem)
			if !ok {
				return m, nil
			}
			if m.mode == ModeSelectTwo {
				if m.selection1 == nil {
					s := selected.snapshot
					m.selection1 = &s
					m.list.Title = "Select second snapshot"
					return m, nil
				}
				s := selected.snapshot
				m.selection2 = &s
				return m, compareSnapshotsCmd(m.selection1, m.selection2)
			}
		case "esc":
			m.selection1 = nil
			m.selection2 = nil
			m.mode = ModeBrowse
			m.list.Title = "Snapshots"
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SnapshotListModel) View() string {
	if len(m.snapshots) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("SNAPSHOT HISTORY"),
			NewAlert(AlertInfo, "No snapshots yet", "Use 'Take New Snapshot' from the home screen.").Render(),
			helpStyle.Render("esc back | q quit"),
		)
	}

	title := titleStyle.Render("SNAPSHOT HISTORY")
	modeLabel := subtitleStyle.Render("mode: " + string(m.mode))
	listContent := panelStyle.Render(m.list.View())

	selectionInfo := "No selection yet"
	if m.selection1 != nil {
		selectionInfo = fmt.Sprintf("selected #1: %s\n", m.selection1.Timestamp.Format("2006-01-02 15:04:05"))
	}
	if m.selection2 != nil {
		selectionInfo += fmt.Sprintf("selected #2: %s\n", m.selection2.Timestamp.Format("2006-01-02 15:04:05"))
	}

	summary := NewTable([]TableColumn{{Header: "Selection", Width: 14}, {Header: "Value", Width: 44}})
	summary.AddRow([]string{"Mode", string(m.mode)})
	summary.AddRow([]string{"Current", TruncateString(selectionInfo, 42)})
	if m.selection1 != nil {
		summary.AddRow([]string{"#1 ID", TruncateString(m.selection1.ID, 42)})
	}
	if m.selection2 != nil {
		summary.AddRow([]string{"#2 ID", TruncateString(m.selection2.ID, 42)})
	}

	help := helpStyle.Render("up/down navigate | / filter | enter select | esc back")
	return lipgloss.JoinVertical(lipgloss.Left, title, modeLabel, "", listContent, summary.Render(), help)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
