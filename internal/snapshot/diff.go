package snapshot

type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "ADDED"
	ChangeTypeDeleted  ChangeType = "DELETED"
	ChangeTypeModified ChangeType = "MODIFIED"
)

type Change struct {
	Type     ChangeType `json:"type"`
	Entity   string     `json:"entity"`
	Field    string     `json:"field"`
	OldValue any        `json:"old_value,omitempty"`
	NewValue any        `json:"new_value,omitempty"`
	Severity string     `json:"severity"`
}

type DiffResult struct {
	Changes        []Change       `json:"changes"`
	AffectedTables []string       `json:"affected_tables"`
	Summary        map[string]int `json:"summary"`
}

func CompareSnapshots(previous, current *StateSnapshot) *DiffResult {
	diff := &DiffResult{
		Changes:        make([]Change, 0),
		AffectedTables: make([]string, 0),
		Summary: map[string]int{
			string(ChangeTypeAdded):    0,
			string(ChangeTypeDeleted):  0,
			string(ChangeTypeModified): 0,
		},
	}

	affectedSet := make(map[string]struct{})

	for fqn := range previous.Tables {
		if _, ok := current.Tables[fqn]; !ok {
			diff.Changes = append(diff.Changes, Change{Type: ChangeTypeDeleted, Entity: fqn, Severity: "CRITICAL"})
			diff.Summary[string(ChangeTypeDeleted)]++
			affectedSet[fqn] = struct{}{}
		}
	}

	for fqn, currentTable := range current.Tables {
		previousTable, ok := previous.Tables[fqn]
		if !ok {
			diff.Changes = append(diff.Changes, Change{Type: ChangeTypeAdded, Entity: fqn, Severity: "INFO"})
			diff.Summary[string(ChangeTypeAdded)]++
			affectedSet[fqn] = struct{}{}
			continue
		}

		tableChanges := compareTableStates(previousTable, currentTable)
		if len(tableChanges) > 0 {
			diff.Changes = append(diff.Changes, tableChanges...)
			diff.Summary[string(ChangeTypeModified)]++
			affectedSet[fqn] = struct{}{}
		}
	}

	for fqn := range affectedSet {
		diff.AffectedTables = append(diff.AffectedTables, fqn)
	}
	return diff
}

func compareTableStates(previous, current *TableState) []Change {
	changes := make([]Change, 0)

	if previous.Description != current.Description {
		changes = append(changes, Change{
			Type:     ChangeTypeModified,
			Entity:   previous.FQN,
			Field:    "description",
			OldValue: previous.Description,
			NewValue: current.Description,
			Severity: "INFO",
		})
	}

	for columnName, previousColumn := range previous.Columns {
		if _, ok := current.Columns[columnName]; !ok {
			changes = append(changes, Change{
				Type:     ChangeTypeDeleted,
				Entity:   previous.FQN,
				Field:    columnName,
				OldValue: previousColumn.DataType,
				Severity: "CRITICAL",
			})
		}
	}

	for columnName, currentColumn := range current.Columns {
		previousColumn, ok := previous.Columns[columnName]
		if !ok {
			changes = append(changes, Change{
				Type:     ChangeTypeAdded,
				Entity:   previous.FQN,
				Field:    columnName,
				NewValue: currentColumn.DataType,
				Severity: "INFO",
			})
			continue
		}

		if previousColumn.DataType != currentColumn.DataType {
			changes = append(changes, Change{
				Type:     ChangeTypeModified,
				Entity:   previous.FQN,
				Field:    columnName,
				OldValue: previousColumn.DataType,
				NewValue: currentColumn.DataType,
				Severity: "WARNING",
			})
		}
		if previousColumn.Nullable != currentColumn.Nullable {
			changes = append(changes, Change{
				Type:     ChangeTypeModified,
				Entity:   previous.FQN,
				Field:    columnName + ".nullable",
				OldValue: previousColumn.Nullable,
				NewValue: currentColumn.Nullable,
				Severity: "WARNING",
			})
		}
	}

	return changes
}
