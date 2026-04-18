package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type guardProfile struct {
	Version   string               `yaml:"version" json:"version"`
	Domain    string               `yaml:"domain" json:"domain"`
	Drift     guardDriftPolicy     `yaml:"drift" json:"drift"`
	Contracts guardContractsPolicy `yaml:"contracts" json:"contracts"`
}

type guardDriftPolicy struct {
	FailOn   string `yaml:"fail_on" json:"fail_on"`
	MaxTotal *int   `yaml:"max_total" json:"max_total"`
}

type guardContractsPolicy struct {
	MinScore                *int  `yaml:"min_score" json:"min_score"`
	RequireOwner            *bool `yaml:"require_owner" json:"require_owner"`
	RequireTableDescription *bool `yaml:"require_table_description" json:"require_table_description"`
}

var guardProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage and validate guard profile files",
}

var guardProfileValidateCmd = &cobra.Command{
	Use:   "validate <guard.yaml>",
	Short: "Validate a guard profile file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := strings.TrimSpace(args[0])
		p, err := loadGuardProfile(path)
		if err != nil {
			return err
		}

		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"valid":   true,
				"profile": path,
				"domain":  p.Domain,
				"version": p.Version,
			})
		}

		fmt.Printf("valid: true\n")
		fmt.Printf("profile: %s\n", path)
		fmt.Printf("domain: %s\n", p.Domain)
		fmt.Printf("version: %s\n", p.Version)
		return nil
	},
}

var guardProfileShowCmd = &cobra.Command{
	Use:   "show <guard.yaml>",
	Short: "Show normalized guard profile content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := strings.TrimSpace(args[0])
		p, err := loadGuardProfile(path)
		if err != nil {
			return err
		}

		if wantsJSONOutput() {
			return printStructured(map[string]any{"profile": p})
		}

		fmt.Printf("domain: %s\n", p.Domain)
		fmt.Printf("version: %s\n", p.Version)
		fmt.Printf("drift.fail_on: %s\n", p.Drift.FailOn)
		if p.Drift.MaxTotal != nil {
			fmt.Printf("drift.max_total: %d\n", *p.Drift.MaxTotal)
		} else {
			fmt.Printf("drift.max_total: (not set)\n")
		}
		if p.Contracts.MinScore != nil {
			fmt.Printf("contracts.min_score: %d\n", *p.Contracts.MinScore)
		} else {
			fmt.Printf("contracts.min_score: (not set)\n")
		}
		if p.Contracts.RequireOwner != nil {
			fmt.Printf("contracts.require_owner: %t\n", *p.Contracts.RequireOwner)
		} else {
			fmt.Printf("contracts.require_owner: (not set)\n")
		}
		if p.Contracts.RequireTableDescription != nil {
			fmt.Printf("contracts.require_table_description: %t\n", *p.Contracts.RequireTableDescription)
		} else {
			fmt.Printf("contracts.require_table_description: (not set)\n")
		}
		return nil
	},
}

func init() {
	guardProfileCmd.AddCommand(guardProfileValidateCmd)
	guardProfileCmd.AddCommand(guardProfileShowCmd)
}

func loadGuardProfile(path string) (*guardProfile, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("profile path is required")
	}
	clean := filepath.Clean(strings.TrimSpace(path))
	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read guard profile: %w", err)
	}

	var p guardProfile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse guard profile yaml: %w", err)
	}
	if err := validateGuardProfile(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func validateGuardProfile(p *guardProfile) error {
	if p == nil {
		return fmt.Errorf("guard profile is nil")
	}
	if strings.TrimSpace(p.Version) == "" {
		p.Version = "v1"
	}
	if !strings.EqualFold(strings.TrimSpace(p.Version), "v1") {
		return fmt.Errorf("unsupported profile version %q (expected v1)", p.Version)
	}
	if strings.TrimSpace(p.Domain) == "" {
		return fmt.Errorf("profile domain is required")
	}

	if strings.TrimSpace(p.Drift.FailOn) != "" {
		if severityRank(p.Drift.FailOn) < 0 {
			return fmt.Errorf("invalid drift.fail_on %q", p.Drift.FailOn)
		}
	}
	if p.Drift.MaxTotal != nil && *p.Drift.MaxTotal < -1 {
		return fmt.Errorf("drift.max_total must be >= -1")
	}

	if p.Contracts.MinScore != nil && (*p.Contracts.MinScore < 0 || *p.Contracts.MinScore > 100) {
		return fmt.Errorf("contracts.min_score must be between 0 and 100")
	}
	return nil
}
