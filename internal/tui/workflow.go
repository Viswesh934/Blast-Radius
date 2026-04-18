package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/Viswesh934/blast-radius/internal/impact"
	"github.com/Viswesh934/blast-radius/internal/snapshot"
)

type SnapshotWorkflow struct {
	currentStep   int
	totalSteps    int
	statusUpdates []string
}

func NewSnapshotWorkflow() *SnapshotWorkflow {
	return &SnapshotWorkflow{currentStep: 0, totalSteps: 3, statusUpdates: []string{}}
}

func (w *SnapshotWorkflow) Execute(_ context.Context, directory string) (*SnapshotInfo, error) {
	w.currentStep = 1
	w.addStatus("Preparing snapshot storage")

	w.currentStep = 2
	w.addStatus("Creating snapshot")
	snap, err := createSnapshot(directory)
	if err != nil {
		return nil, err
	}

	w.currentStep = 3
	w.addStatus("Snapshot completed")
	return snap, nil
}

func (w *SnapshotWorkflow) GetProgress() float64 {
	return float64(w.currentStep) / float64(max(1, w.totalSteps))
}

func (w *SnapshotWorkflow) GetStatusUpdates() []string {
	return w.statusUpdates
}

func (w *SnapshotWorkflow) addStatus(status string) {
	w.statusUpdates = append(w.statusUpdates, status)
}

type ComparisonWorkflow struct {
	snap1         *SnapshotInfo
	snap2         *SnapshotInfo
	currentStep   int
	totalSteps    int
	statusUpdates []string
}

func NewComparisonWorkflow(snap1, snap2 *SnapshotInfo) *ComparisonWorkflow {
	return &ComparisonWorkflow{snap1: snap1, snap2: snap2, currentStep: 0, totalSteps: 3, statusUpdates: []string{}}
}

func (w *ComparisonWorkflow) Execute(_ context.Context) (*snapshot.DiffResult, error) {
	if w.snap1 == nil || w.snap2 == nil {
		return nil, fmt.Errorf("two snapshots are required")
	}

	w.currentStep = 1
	w.addStatus("Loading snapshots")
	left, err := snapshot.Load(w.snap1.Path)
	if err != nil {
		return nil, err
	}
	right, err := snapshot.Load(w.snap2.Path)
	if err != nil {
		return nil, err
	}

	w.currentStep = 2
	w.addStatus("Computing differences")
	diff := snapshot.CompareSnapshots(left, right)

	w.currentStep = 3
	w.addStatus(fmt.Sprintf("Found %d changes", len(diff.Changes)))
	return diff, nil
}

func (w *ComparisonWorkflow) GetProgress() float64 {
	return float64(w.currentStep) / float64(max(1, w.totalSteps))
}

func (w *ComparisonWorkflow) GetStatusUpdates() []string {
	return w.statusUpdates
}

func (w *ComparisonWorkflow) addStatus(status string) {
	w.statusUpdates = append(w.statusUpdates, status)
}

type ImpactWorkflow struct {
	snap1         *SnapshotInfo
	snap2         *SnapshotInfo
	changes       []snapshot.Change
	currentStep   int
	totalSteps    int
	statusUpdates []string
}

func NewImpactWorkflow(snap1, snap2 *SnapshotInfo, changes []snapshot.Change) *ImpactWorkflow {
	return &ImpactWorkflow{snap1: snap1, snap2: snap2, changes: changes, currentStep: 0, totalSteps: 4, statusUpdates: []string{}}
}

func (w *ImpactWorkflow) Execute(_ context.Context) (*ImpactResult, error) {
	if w.snap1 == nil || w.snap2 == nil {
		return nil, fmt.Errorf("two snapshots are required")
	}

	start := time.Now()
	w.currentStep = 1
	w.addStatus("Loading snapshots")
	_, err := snapshot.Load(w.snap1.Path)
	if err != nil {
		return nil, err
	}
	right, err := snapshot.Load(w.snap2.Path)
	if err != nil {
		return nil, err
	}

	w.currentStep = 2
	w.addStatus("Normalizing change data")
	if len(w.changes) == 0 {
		diff := snapshot.CompareSnapshots(right, right)
		w.changes = diff.Changes
	}

	w.currentStep = 3
	w.addStatus("Computing impact score")
	analysis := impact.AnalyzeImpact(right, w.changes)

	w.currentStep = 4
	w.addStatus("Impact workflow complete")

	result := &ImpactResult{
		Changes:         toTUIChanges(analysis.Changes),
		ImpactedAssets:  toTUIAssets(analysis.ImpactedAssets),
		RiskLevel:       analysis.RiskLevel,
		Recommendations: analysis.RecommendedActions,
		ExecutionTime:   time.Since(start),
	}
	return result, nil
}

func (w *ImpactWorkflow) GetProgress() float64 {
	return float64(w.currentStep) / float64(max(1, w.totalSteps))
}

func (w *ImpactWorkflow) GetStatusUpdates() []string {
	return w.statusUpdates
}

func (w *ImpactWorkflow) addStatus(status string) {
	w.statusUpdates = append(w.statusUpdates, status)
}

func toTUIChanges(changes []snapshot.Change) []Change {
	out := make([]Change, 0, len(changes))
	for _, c := range changes {
		out = append(out, Change{
			Type:     string(c.Type),
			Entity:   c.Entity,
			Field:    c.Field,
			OldValue: c.OldValue,
			NewValue: c.NewValue,
			Severity: c.Severity,
		})
	}
	return out
}

func toTUIAssets(assets []impact.ImpactedAsset) []ImpactedAsset {
	out := make([]ImpactedAsset, 0, len(assets))
	for _, a := range assets {
		out = append(out, ImpactedAsset{
			FQN:              a.FQN,
			Type:             a.Type,
			Reason:           a.Reason,
			DirectlyAffected: a.DirectlyAffected,
			DownstreamCount:  0,
		})
	}
	return out
}

type ProgressScreen struct {
	title      string
	workflow   interface{}
	statusLogs []string
	startTime  time.Time
}

func NewProgressScreen(title string, workflow interface{}) ProgressScreen {
	return ProgressScreen{title: title, workflow: workflow, startTime: time.Now(), statusLogs: []string{}}
}

func (p ProgressScreen) View() string {
	progress := 0.0
	switch w := p.workflow.(type) {
	case *SnapshotWorkflow:
		progress = w.GetProgress()
		p.statusLogs = w.GetStatusUpdates()
	case *ComparisonWorkflow:
		progress = w.GetProgress()
		p.statusLogs = w.GetStatusUpdates()
	case *ImpactWorkflow:
		progress = w.GetProgress()
		p.statusLogs = w.GetStatusUpdates()
	}

	bar := NewProgressBar("progress", 100)
	bar.Current = int(progress * 100)

	lines := []string{Styles.Title.Render(p.title), bar.Render(), "", Styles.Label.Render("Status")}
	for _, s := range p.statusLogs {
		lines = append(lines, "  + "+Styles.Muted.Render(s))
	}
	lines = append(lines, "", Styles.Muted.Render(fmt.Sprintf("Elapsed: %s", time.Since(p.startTime).Round(100*time.Millisecond))))
	return JoinColumns(lines...)
}
