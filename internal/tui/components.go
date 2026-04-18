package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var Colors = struct {
	Primary    string
	Secondary  string
	Accent     string
	Background string
	Foreground string
	Success    string
	Error      string
	Warning    string
	Info       string
	Critical   string
	Border     string
}{
	Primary:    "205",
	Secondary:  "63",
	Accent:     "226",
	Background: "235",
	Foreground: "255",
	Success:    "82",
	Error:      "196",
	Warning:    "208",
	Info:       "39",
	Critical:   "196",
	Border:     "240",
}

var Styles = struct {
	Title           lipgloss.Style
	Subtitle        lipgloss.Style
	Label           lipgloss.Style
	Value           lipgloss.Style
	Success         lipgloss.Style
	Error           lipgloss.Style
	Warning         lipgloss.Style
	Info            lipgloss.Style
	Border          lipgloss.Style
	BorderHighlight lipgloss.Style
	Muted           lipgloss.Style
}{
	Title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Colors.Primary)).MarginBottom(1),
	Subtitle: lipgloss.NewStyle().
		Foreground(lipgloss.Color(Colors.Border)).
		MarginBottom(1),
	Label:   lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Border)).Bold(true),
	Value:   lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Foreground)).Bold(true),
	Success: lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Success)).Bold(true),
	Error:   lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Error)).Bold(true),
	Warning: lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Warning)).Bold(true),
	Info:    lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Info)).Bold(true),
	Border: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(Colors.Border)).
		Padding(1).
		MarginBottom(1),
	BorderHighlight: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(Colors.Accent)).
		Padding(1).
		MarginBottom(1),
	Muted: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
}

type Badge struct {
	Text    string
	Color   string
	BgColor string
	Bold    bool
}

func NewBadge(text string, color string) Badge {
	return Badge{Text: text, Color: color, BgColor: "0", Bold: true}
}

func (b Badge) Render() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(b.Color)).Padding(0, 1)
	if b.BgColor != "0" {
		style = style.Background(lipgloss.Color(b.BgColor))
	}
	if b.Bold {
		style = style.Bold(true)
	}
	return style.Render(b.Text)
}

func BadgeCritical(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1).Bold(true).Render("CRIT " + text)
}

func BadgeHigh(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("208")).Padding(0, 1).Bold(true).Render("HIGH " + text)
}

func BadgeMedium(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("226")).Padding(0, 1).Bold(true).Render("MED " + text)
}

func BadgeLow(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("82")).Padding(0, 1).Bold(true).Render("LOW " + text)
}

type Button struct {
	Label      string
	IsSelected bool
	IsDisabled bool
}

func NewButton(label string) Button {
	return Button{Label: label}
}

func (b Button) Render() string {
	style := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(Colors.Border))
	if b.IsSelected {
		style = style.Foreground(lipgloss.Color(Colors.Background)).Background(lipgloss.Color(Colors.Accent)).Bold(true)
	} else if b.IsDisabled {
		style = style.Foreground(lipgloss.Color(Colors.Border)).Strikethrough(true)
	} else {
		style = style.Foreground(lipgloss.Color(Colors.Foreground))
	}
	return style.Render(b.Label)
}

type StatBox struct {
	Label string
	Value string
	Icon  string
	Color string
}

func NewStatBox(label, value, icon, color string) StatBox {
	return StatBox{Label: label, Value: value, Icon: icon, Color: color}
}

func (s StatBox) Render(width int) string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Border)).Bold(true).Render(s.Icon+" "+s.Label),
		lipgloss.NewStyle().Foreground(lipgloss.Color(s.Color)).Bold(true).Render(s.Value),
	)
	return Styles.Border.BorderForeground(lipgloss.Color(s.Color)).Width(width).Render(content)
}

type TableColumn struct {
	Header string
	Width  int
}

type TableRow struct {
	Cells []string
	Style lipgloss.Style
}

type Table struct {
	Columns []TableColumn
	Rows    []TableRow
}

func NewTable(columns []TableColumn) Table {
	return Table{Columns: columns, Rows: []TableRow{}}
}

func (t *Table) AddRow(cells []string) {
	t.Rows = append(t.Rows, TableRow{Cells: cells, Style: lipgloss.NewStyle()})
}

func (t Table) Render() string {
	headerCells := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		headerCells[i] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Colors.Accent)).Width(col.Width).Render(col.Header)
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)

	rows := make([]string, 0, len(t.Rows))
	for _, row := range t.Rows {
		rowCells := make([]string, len(t.Columns))
		for i := range t.Columns {
			cell := ""
			if i < len(row.Cells) {
				cell = row.Cells[i]
			}
			rowCells[i] = lipgloss.NewStyle().Width(t.Columns[i].Width).Render(cell)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCells...))
	}
	return Styles.Border.Render(lipgloss.JoinVertical(lipgloss.Left, append([]string{header}, rows...)...))
}

type ProgressBar struct {
	Current int
	Total   int
	Label   string
	Width   int
}

func NewProgressBar(label string, total int) ProgressBar {
	return ProgressBar{Current: 0, Total: max(1, total), Label: label, Width: 30}
}

func (p ProgressBar) Render() string {
	total := max(1, p.Total)
	current := p.Current
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(p.Width))

	bar := "["
	for i := 0; i < p.Width; i++ {
		if i < filled {
			bar += "="
		} else {
			bar += "-"
		}
	}
	bar += "]"

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Border)).Render(p.Label+" "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Success)).Render(bar),
		lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Border)).Render(fmt.Sprintf(" %d/%d", current, total)),
	)
}

type Alert struct {
	Type    AlertType
	Title   string
	Message string
}

type AlertType string

const (
	AlertSuccess AlertType = "success"
	AlertError   AlertType = "error"
	AlertWarning AlertType = "warning"
	AlertInfo    AlertType = "info"
)

func NewAlert(alertType AlertType, title, message string) Alert {
	return Alert{Type: alertType, Title: title, Message: message}
}

func (a Alert) Render() string {
	icon := "i"
	color := Colors.Info
	switch a.Type {
	case AlertSuccess:
		icon = "+"
		color = Colors.Success
	case AlertError:
		icon = "x"
		color = Colors.Error
	case AlertWarning:
		icon = "!"
		color = Colors.Warning
	}
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(icon+" "+a.Title),
		lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Foreground)).Render(a.Message),
	)
	return Styles.Border.BorderForeground(lipgloss.Color(color)).Render(content)
}

type Spinner struct {
	frames []string
	index  int
	label  string
}

func NewSpinner(label string) Spinner {
	return Spinner{frames: []string{"-", "\\", "|", "/"}, label: label}
}

func (s *Spinner) Next() string {
	frame := s.frames[s.index%len(s.frames)]
	s.index++
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Primary)).Render(frame + " " + s.label)
}

type TreeNode struct {
	Label    string
	Children []*TreeNode
	Icon     string
	Style    lipgloss.Style
}

func NewTreeNode(label, icon string) *TreeNode {
	return &TreeNode{Label: label, Icon: icon, Children: []*TreeNode{}, Style: lipgloss.NewStyle()}
}

func (t *TreeNode) Add(child *TreeNode) {
	t.Children = append(t.Children, child)
}

func (t *TreeNode) Render(indent string, isLast bool) string {
	prefix := "|- "
	if isLast {
		prefix = "`- "
	}
	line := indent + prefix + t.Icon + " " + t.Label + "\n"
	for i, child := range t.Children {
		childIndent := indent
		if isLast {
			childIndent += "   "
		} else {
			childIndent += "|  "
		}
		line += child.Render(childIndent, i == len(t.Children)-1)
	}
	return line
}

func TruncateString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxWidth {
		return s
	}
	r := []rune(s)
	if maxWidth <= 3 {
		return string(r[:maxWidth])
	}
	return string(r[:maxWidth-3]) + "..."
}

func PadString(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

func CenterString(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(s)
}

func JoinRows(items ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, items...)
}

func JoinColumns(items ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

func RiskBadge(riskLevel string) string {
	switch strings.ToUpper(riskLevel) {
	case "CRITICAL":
		return BadgeCritical("CRITICAL")
	case "HIGH":
		return BadgeHigh("HIGH")
	case "MEDIUM":
		return BadgeMedium("MEDIUM")
	default:
		return BadgeLow("LOW")
	}
}

type Dialog struct {
	Title       string
	Message     string
	Options     []string
	SelectedIdx int
	IsOpen      bool
}

func NewDialog(title, message string, options []string) Dialog {
	return Dialog{Title: title, Message: message, Options: options, SelectedIdx: 0, IsOpen: true}
}

func (d Dialog) Render() string {
	var options []string
	for i, option := range d.Options {
		style := lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color(Colors.Foreground))
		if i == d.SelectedIdx {
			style = style.Background(lipgloss.Color(Colors.Accent)).Foreground(lipgloss.Color(Colors.Background)).Bold(true)
		}
		options = append(options, style.Render(option))
	}
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Colors.Primary)).MarginBottom(1).Render(d.Title),
		lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Foreground)).MarginBottom(1).Render(d.Message),
		strings.Join(options, " "),
	)
	return lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color(Colors.Primary)).Padding(1).Render(content)
}

type TimelineEvent struct {
	Time    string
	Title   string
	Details string
	Status  string
}

type Timeline struct {
	Events []TimelineEvent
}

func NewTimeline() Timeline {
	return Timeline{Events: []TimelineEvent{}}
}

func (t *Timeline) AddEvent(event TimelineEvent) {
	t.Events = append(t.Events, event)
}

func (t Timeline) Render() string {
	var out strings.Builder
	for i, event := range t.Events {
		icon := "*"
		color := Colors.Info
		switch event.Status {
		case "success":
			icon = "+"
			color = Colors.Success
		case "error":
			icon = "x"
			color = Colors.Error
		case "pending":
			icon = "o"
			color = Colors.Warning
		}
		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Border)).Width(12).Render(event.Time),
			lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(icon),
			lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Foreground)).Bold(true).Render(" "+event.Title),
		)
		out.WriteString(line + "\n")
		if event.Details != "" {
			out.WriteString("             " + lipgloss.NewStyle().Foreground(lipgloss.Color(Colors.Border)).Render(event.Details) + "\n")
		}
		if i < len(t.Events)-1 {
			out.WriteString("             |\n")
		}
	}
	return out.String()
}
