package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Save(snap *StateSnapshot, directory string) (string, error) {
	payload, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(directory, fmt.Sprintf("snapshot_%d.json", snap.Timestamp.Unix()))
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func Load(path string) (*StateSnapshot, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap StateSnapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return nil, err
	}
	if snap.Tables == nil {
		snap.Tables = map[string]*TableState{}
	}
	if snap.Lineage == nil {
		snap.Lineage = map[string]*LineageState{}
	}
	if snap.Metadata == nil {
		snap.Metadata = map[string]string{}
	}
	return &snap, nil
}
