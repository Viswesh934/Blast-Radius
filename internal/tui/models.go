package tui

import (
	"time"

	"github.com/Viswesh934/blast-radius/internal/config"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Screen string

const (
	HomeScreen         Screen = "home"
	SnapshotListScreen Screen = "snapshot_list"
	ImpactScreen       Screen = "impact"
	LoadingScreen      Screen = "loading"
)

type SelectionMode string

const (
	ModeBrowse    SelectionMode = "browse"
	ModeSelectTwo SelectionMode = "select_two"
)

type MessageType string

const (
	MessageSuccess MessageType = "success"
	MessageError   MessageType = "error"
	MessageInfo    MessageType = "info"
	MessageWarning MessageType = "warning"
)

type Model struct {
	currentScreen Screen
	width         int
	height        int

	cfg *config.Config

	snapshots []SnapshotInfo
	selected1 *SnapshotInfo
	selected2 *SnapshotInfo

	loading     bool
	message     string
	messageType MessageType
	helpVisible bool

	home         homeModel
	snapshotList SnapshotListModel
	impactView   ImpactModel
	impactResult *ImpactResult
	spinner      spinner.Model
}

type SnapshotInfo struct {
	ID           string
	Timestamp    time.Time
	Source       string
	TableCount   int
	LineageCount int
	Hash         string
	Path         string
}

type ImpactResult struct {
	Changes         []Change
	ImpactedAssets  []ImpactedAsset
	RiskLevel       string
	Recommendations []string
	ExecutionTime   time.Duration
}

type Change struct {
	Type     string
	Entity   string
	Field    string
	OldValue interface{}
	NewValue interface{}
	Severity string
}

type ImpactedAsset struct {
	FQN              string
	Type             string
	Reason           string
	DirectlyAffected bool
	DownstreamCount  int
}

type SnapshotCreatedMsg struct {
	Snapshot *SnapshotInfo
	Error    error
}

type SnapshotsLoadedMsg struct {
	Snapshots []SnapshotInfo
	Error     error
}

type ComparisonCompleteMsg struct {
	Snapshot1 *SnapshotInfo
	Snapshot2 *SnapshotInfo
	Changes   []Change
	Error     error
}

type ImpactAnalyzedMsg struct {
	Result *ImpactResult
	Error  error
}

type MessageMsg struct {
	Text     string
	Type     MessageType
	Duration time.Duration
}

type BackToHomeMsg struct{}

type StartCompareModeMsg struct{}

type StartBrowseModeMsg struct{}

type AnalyzeSelectedMsg struct{}

type MenuItem struct {
	Title       string
	Description string
	Icon        string
	Action      func(*Model) tea.Cmd
}

type snapshotItem struct {
	snapshot SnapshotInfo
}

func (i snapshotItem) Title() string {
	return i.snapshot.Timestamp.Format("2006-01-02 15:04:05") + "  |  " + i.snapshot.Source
}

func (i snapshotItem) Description() string {
	return "tables=" + itoa(i.snapshot.TableCount) + " lineage=" + itoa(i.snapshot.LineageCount) + " id=" + i.snapshot.ID
}

func (i snapshotItem) FilterValue() string {
	return i.snapshot.ID + " " + i.snapshot.Source
}

type SnapshotListModel struct {
	list       list.Model
	snapshots  []SnapshotInfo
	mode       SelectionMode
	selection1 *SnapshotInfo
	selection2 *SnapshotInfo
}

type ImpactModel struct {
	result        *ImpactResult
	scrollPos     int
	selectedAsset int
}

func itoa(v int) string {
	return fmtInt(v)
}
