package tui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func NewModel() Model {
	cfg, err := config.Load("")
	if err != nil {
		cfg = &config.Config{}
	}
	if cfg.Snapshot.Directory == "" {
		cfg.Snapshot.Directory = "./snapshots"
	}

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = accentStyle

	m := Model{
		currentScreen: HomeScreen,
		cfg:           cfg,
		snapshots:     []SnapshotInfo{},
		helpVisible:   false,
		spinner:       sp,
	}
	m.home = newHomeModel(&m)
	m.snapshotList = NewSnapshotListModel(nil)
	return m
}

func Run() error {
	model := NewModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSnapshotsCmd(m.cfg.Snapshot.Directory), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.snapshotList, _ = m.snapshotList.Update(msg)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.helpVisible = !m.helpVisible
			return m, nil
		case "esc":
			if m.currentScreen != HomeScreen {
				m.currentScreen = HomeScreen
				return m, nil
			}
		}

	case spinner.TickMsg:
		if m.currentScreen == LoadingScreen {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case StartCompareModeMsg:
		m.currentScreen = SnapshotListScreen
		m.snapshotList = NewSnapshotListModel(m.snapshots)
		m.snapshotList.mode = ModeSelectTwo
		m.snapshotList.list.Title = "Select first snapshot"
		m.message = "Pick snapshot #1, then snapshot #2"
		m.messageType = MessageInfo
		return m, nil

	case StartBrowseModeMsg:
		m.currentScreen = SnapshotListScreen
		m.snapshotList = NewSnapshotListModel(m.snapshots)
		m.snapshotList.mode = ModeBrowse
		m.snapshotList.list.Title = "Snapshot history"
		m.message = "Browse snapshots"
		m.messageType = MessageInfo
		return m, nil

	case BackToHomeMsg:
		m.currentScreen = HomeScreen
		return m, nil

	case MessageMsg:
		m.message = msg.Text
		m.messageType = msg.Type
		if msg.Duration > 0 {
			return m, tea.Tick(msg.Duration, func(time.Time) tea.Msg {
				return MessageMsg{Text: "", Type: MessageInfo}
			})
		}
		return m, nil

	case SnapshotsLoadedMsg:
		if msg.Error != nil {
			m.message = "Error loading snapshots: " + msg.Error.Error()
			m.messageType = MessageError
			return m, nil
		}
		m.snapshots = msg.Snapshots
		m.home.model = &m
		m.snapshotList = NewSnapshotListModel(m.snapshots)
		if m.currentScreen == SnapshotListScreen {
			if m.snapshotList.mode == ModeSelectTwo {
				m.snapshotList.list.Title = "Select first snapshot"
			} else {
				m.snapshotList.list.Title = "Snapshot history"
			}
		}
		return m, nil

	case SnapshotCreatedMsg:
		m.loading = false
		m.currentScreen = HomeScreen
		if msg.Error != nil {
			m.message = "Error creating snapshot: " + msg.Error.Error()
			m.messageType = MessageError
			return m, nil
		}
		if msg.Snapshot != nil {
			m.snapshots = append([]SnapshotInfo{*msg.Snapshot}, m.snapshots...)
			m.message = "Snapshot created: " + msg.Snapshot.ID
			m.messageType = MessageSuccess
		}
		m.home.model = &m
		return m, nil

	case ComparisonCompleteMsg:
		if msg.Error != nil {
			m.currentScreen = SnapshotListScreen
			m.message = "Compare failed: " + msg.Error.Error()
			m.messageType = MessageError
			return m, nil
		}
		m.currentScreen = LoadingScreen
		m.loading = true
		m.message = "Analyzing downstream impact..."
		m.messageType = MessageInfo
		return m, tea.Batch(analyzeImpactCmd(msg.Snapshot1, msg.Snapshot2), m.spinner.Tick)

	case ImpactAnalyzedMsg:
		m.loading = false
		if msg.Error != nil {
			m.currentScreen = HomeScreen
			m.message = "Impact analysis failed: " + msg.Error.Error()
			m.messageType = MessageError
			return m, nil
		}
		m.impactResult = msg.Result
		m.impactView = NewImpactModel(msg.Result)
		m.currentScreen = ImpactScreen
		m.message = "Impact analysis complete"
		m.messageType = MessageSuccess
		return m, nil
	}

	switch m.currentScreen {
	case HomeScreen:
		h, cmd := m.home.Update(msg)
		m.home = h
		return m, cmd
	case SnapshotListScreen:
		sl, cmd := m.snapshotList.Update(msg)
		m.snapshotList = sl
		return m, cmd
	case ImpactScreen:
		iv, cmd := m.impactView.Update(msg)
		m.impactView = iv
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var content string
	switch m.currentScreen {
	case HomeScreen:
		content = m.home.View()
	case SnapshotListScreen:
		content = m.snapshotList.View()
	case ImpactScreen:
		content = m.impactView.View()
	case LoadingScreen:
		content = m.renderLoading()
	default:
		content = "Unknown screen"
	}

	view := lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), content, m.renderMessageBar())
	view = docStyle.Render(view)
	if m.helpVisible {
		view = lipgloss.JoinVertical(lipgloss.Left, view, "", m.renderHelpOverlay())
	}
	return view
}

func (m Model) renderHeader() string {
	width := m.width
	if width <= 0 {
		width = 100
	}
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(Colors.Secondary)).
		Foreground(lipgloss.Color(Colors.Foreground)).
		Bold(true).
		Padding(0, 1).
		Width(width - 6)
	text := fmt.Sprintf("BLAST RADIUS  |  snapshots=%d  |  screen=%s  |  ? help", len(m.snapshots), m.currentScreen)
	return headerStyle.Render(text)
}

func (m Model) renderMessageBar() string {
	if m.message == "" {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Padding(0, 1)
	switch m.messageType {
	case MessageSuccess:
		style = style.Background(lipgloss.Color(Colors.Success))
	case MessageError:
		style = style.Background(lipgloss.Color(Colors.Error))
	case MessageWarning:
		style = style.Background(lipgloss.Color(Colors.Warning))
	default:
		style = style.Background(lipgloss.Color(Colors.Border))
	}
	return style.Render(m.message)
}

func (m Model) renderLoading() string {
	line := m.spinner.View() + " working..."
	if m.message != "" {
		line = m.spinner.View() + " " + m.message
	}
	return panelStyle.Render(line)
}

func (m Model) renderHelpOverlay() string {
	helpText := strings.Join([]string{
		"up/down or j/k: navigate",
		"left/right or h/l: move selection on impact screen",
		"enter: select",
		"esc: back",
		"q or ctrl+c: quit",
		"/: filter in snapshot list",
	}, "\n")
	return NewAlert(AlertInfo, "Help", helpText).Render()
}

func loadSnapshotsCmd(directory string) tea.Cmd {
	return func() tea.Msg {
		snapshots, err := loadSnapshotsFromDisk(directory)
		return SnapshotsLoadedMsg{Snapshots: snapshots, Error: err}
	}
}

func captureSnapshotCmd(directory string) tea.Cmd {
	return func() tea.Msg {
		snap, err := createSnapshot(directory)
		return SnapshotCreatedMsg{Snapshot: snap, Error: err}
	}
}

func compareSnapshotsCmd(snap1, snap2 *SnapshotInfo) tea.Cmd {
	return func() tea.Msg {
		changes, err := compareSnapshots(snap1, snap2)
		return ComparisonCompleteMsg{Snapshot1: snap1, Snapshot2: snap2, Changes: changes, Error: err}
	}
}

func analyzeImpactCmd(snap1, snap2 *SnapshotInfo) tea.Cmd {
	return func() tea.Msg {
		result, err := analyzeImpact(snap1, snap2)
		return ImpactAnalyzedMsg{Result: result, Error: err}
	}
}

func loadSnapshotsFromDisk(directory string) ([]SnapshotInfo, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	infos := make([]SnapshotInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "snapshot_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		fullPath := filepath.Join(directory, name)
		s, err := snapshot.Load(fullPath)
		if err != nil {
			continue
		}
		h := sha256.Sum256([]byte(s.ID + s.Timestamp.UTC().String() + name))
		infos = append(infos, SnapshotInfo{
			ID:           s.ID,
			Timestamp:    s.Timestamp,
			Source:       s.Source,
			TableCount:   len(s.Tables),
			LineageCount: len(s.Lineage),
			Hash:         hex.EncodeToString(h[:8]),
			Path:         fullPath,
		})
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].Timestamp.After(infos[j].Timestamp) })
	return infos, nil
}

func createSnapshot(directory string) (*SnapshotInfo, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	s := &snapshot.StateSnapshot{
		ID:        fmt.Sprintf("snap_%d", now.UnixNano()),
		Timestamp: now,
		Source:    "manual.local",
		Tables:    map[string]*snapshot.TableState{},
		Lineage:   map[string]*snapshot.LineageState{},
		Metadata:  map[string]string{"origin": "tui-placeholder"},
	}
	path, err := snapshot.Save(s, directory)
	if err != nil {
		return nil, err
	}

	h := sha256.Sum256([]byte(s.ID + s.Timestamp.UTC().String()))
	return &SnapshotInfo{
		ID:           s.ID,
		Timestamp:    s.Timestamp,
		Source:       s.Source,
		TableCount:   len(s.Tables),
		LineageCount: len(s.Lineage),
		Hash:         hex.EncodeToString(h[:8]),
		Path:         path,
	}, nil
}

func compareSnapshots(snap1, snap2 *SnapshotInfo) ([]Change, error) {
	if snap1 == nil || snap2 == nil {
		return nil, fmt.Errorf("two snapshots are required")
	}
	left, err := snapshot.Load(snap1.Path)
	if err != nil {
		return nil, err
	}
	right, err := snapshot.Load(snap2.Path)
	if err != nil {
		return nil, err
	}
	res := snapshot.CompareSnapshots(left, right)
	changes := make([]Change, 0, len(res.Changes))
	for _, c := range res.Changes {
		changes = append(changes, Change{
			Type:     string(c.Type),
			Entity:   c.Entity,
			Field:    c.Field,
			OldValue: c.OldValue,
			NewValue: c.NewValue,
			Severity: c.Severity,
		})
	}
	return changes, nil
}

func analyzeImpact(snap1, snap2 *SnapshotInfo) (*ImpactResult, error) {
	start := time.Now()
	if snap1 == nil || snap2 == nil {
		return nil, fmt.Errorf("two snapshots are required")
	}
	left, err := snapshot.Load(snap1.Path)
	if err != nil {
		return nil, err
	}
	right, err := snapshot.Load(snap2.Path)
	if err != nil {
		return nil, err
	}
	diff := snapshot.CompareSnapshots(left, right)
	analysis := impact.AnalyzeImpact(right, diff.Changes)

	changes := make([]Change, 0, len(analysis.Changes))
	for _, c := range analysis.Changes {
		changes = append(changes, Change{
			Type:     string(c.Type),
			Entity:   c.Entity,
			Field:    c.Field,
			OldValue: c.OldValue,
			NewValue: c.NewValue,
			Severity: c.Severity,
		})
	}

	assets := make([]ImpactedAsset, 0, len(analysis.ImpactedAssets))
	for _, a := range analysis.ImpactedAssets {
		assets = append(assets, ImpactedAsset{
			FQN:              a.FQN,
			Type:             a.Type,
			Reason:           a.Reason,
			DirectlyAffected: a.DirectlyAffected,
			DownstreamCount:  0,
		})
	}

	return &ImpactResult{
		Changes:         changes,
		ImpactedAssets:  assets,
		RiskLevel:       analysis.RiskLevel,
		Recommendations: analysis.RecommendedActions,
		ExecutionTime:   time.Since(start),
	}, nil
}
