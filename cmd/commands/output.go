package commands

import (
	"encoding/json"
	"fmt"
	"strings"
)

func wantsJSONOutput() bool {
	if cfg == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cfg.Output.Format), "json")
}

func printStructured(payload any, textLines ...string) error {
	if wantsJSONOutput() {
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal output json: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	for _, line := range textLines {
		fmt.Println(line)
	}
	return nil
}
