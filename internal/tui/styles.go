package tui

import "github.com/charmbracelet/lipgloss"

var (
	docStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle    = Styles.Title
	subtitleStyle = Styles.Subtitle
	accentStyle   = Styles.Info
	helpStyle     = Styles.Muted

	menuItemStyle = lipgloss.NewStyle().Padding(0, 1)

	selectedMenuItemStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Bold(true).
				Foreground(lipgloss.Color(Colors.Foreground)).
				Background(lipgloss.Color(Colors.Secondary))

	panelStyle = Styles.Border

	changeDeletedStyle  = Styles.Error
	changeAddedStyle    = Styles.Success
	changeModifiedStyle = Styles.Warning
)
