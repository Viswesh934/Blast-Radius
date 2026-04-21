package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Viswesh934/blast-radius/pkg/netfetch"
)

func main() {
	fetchClient := netfetch.NewClient(10 * time.Second)

	type repoSummary struct {
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		Stargazers    int    `json:"stargazers_count"`
		Forks         int    `json:"forks_count"`
		OpenIssues    int    `json:"open_issues_count"`
		DefaultBranch string `json:"default_branch"`
		UpdatedAt     string `json:"updated_at"`
		HTMLURL       string `json:"html_url"`
	}

	var repo repoSummary
	_, err := fetchClient.GetJSON(
		context.Background(),
		"https://api.github.com/repos/Viswesh934/Blast-Radius",
		map[string]string{"User-Agent": "blast-radius-fetch-demo"},
		nil,
		&repo,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch failed: %v\n", err)
		os.Exit(1)
	}

	output := map[string]any{
		"source":         "github",
		"repository":     repo.FullName,
		"description":    repo.Description,
		"stars":          repo.Stargazers,
		"forks":          repo.Forks,
		"open_issues":    repo.OpenIssues,
		"default_branch": repo.DefaultBranch,
		"updated_at":     repo.UpdatedAt,
		"url":            repo.HTMLURL,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
