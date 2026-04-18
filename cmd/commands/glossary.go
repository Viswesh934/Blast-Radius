package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	glossaryName        string
	glossaryDescription string
	termGlossary        string
	termParent          string
	termName            string
	termDescription     string
)

var glossaryCmd = &cobra.Command{
	Use:     "glossary",
	Aliases: []string{"glossaries"},
	Short:   "Manage OpenMetadata glossaries and glossary terms",
}

var glossaryListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List glossaries and glossary terms",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newOMClient()
		if err != nil {
			return err
		}

		if strings.TrimSpace(glossaryName) == "" {
			glossaries, err := client.ListGlossaries(context.Background())
			if err != nil {
				return fmt.Errorf("list glossaries: %w", err)
			}
			if wantsJSONOutput() {
				return printStructured(map[string]any{
					"glossary_count": len(glossaries),
					"glossaries":     glossaries,
				})
			}
			fmt.Printf("glossary_count: %d\n", len(glossaries))
			for _, glossary := range glossaries {
				fqn := glossary.FullyQualifiedName
				if fqn == "" {
					fqn = glossary.Name
				}
				fmt.Printf("- %s\n", fqn)
			}
			return nil
		}

		terms, err := client.ListGlossaryTerms(context.Background(), glossaryName)
		if err != nil {
			return fmt.Errorf("list glossary terms: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{
				"glossary":   glossaryName,
				"term_count": len(terms),
				"terms":      terms,
			})
		}

		fmt.Printf("glossary: %s\n", glossaryName)
		fmt.Printf("term_count: %d\n", len(terms))
		for _, term := range terms {
			fqn := term.FullyQualifiedName
			if fqn == "" {
				fqn = term.Name
			}
			fmt.Printf("- %s\n", fqn)
		}
		return nil
	},
}

var glossaryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a glossary",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(glossaryName) == "" {
			return fmt.Errorf("--name is required")
		}

		client, err := newOMClient()
		if err != nil {
			return err
		}

		created, err := client.CreateGlossary(context.Background(), glossaryName, glossaryDescription)
		if err != nil {
			return fmt.Errorf("create glossary: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"glossary": created})
		}

		fqn := created.FullyQualifiedName
		if fqn == "" {
			fqn = created.Name
		}
		fmt.Printf("created_glossary: %s\n", fqn)
		fmt.Printf("glossary_id: %s\n", created.ID)
		return nil
	},
}

var glossaryTermCmd = &cobra.Command{
	Use:     "term",
	Aliases: []string{"terms"},
	Short:   "Manage glossary terms",
}

var glossaryTermCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a glossary term",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(termName) == "" {
			return fmt.Errorf("--name is required")
		}
		if strings.TrimSpace(termGlossary) == "" {
			return fmt.Errorf("--glossary is required")
		}

		client, err := newOMClient()
		if err != nil {
			return err
		}

		created, err := client.CreateGlossaryTerm(context.Background(), termGlossary, termParent, termName, termDescription)
		if err != nil {
			return fmt.Errorf("create glossary term: %w", err)
		}
		if wantsJSONOutput() {
			return printStructured(map[string]any{"term": created})
		}

		fqn := created.FullyQualifiedName
		if fqn == "" {
			fqn = created.Name
		}
		fmt.Printf("created_glossary_term: %s\n", fqn)
		fmt.Printf("term_id: %s\n", created.ID)
		fmt.Printf("glossary: %s\n", created.Glossary.FullyQualifiedName)
		return nil
	},
}

func init() {
	glossaryCmd.Flags().StringVar(&glossaryName, "name", "", "glossary name")
	glossaryCmd.Flags().StringVar(&glossaryDescription, "description", "", "glossary description")
	glossaryListCmd.Flags().StringVar(&glossaryName, "glossary", "", "list terms for a specific glossary name/FQN")

	glossaryTermCreateCmd.Flags().StringVar(&termGlossary, "glossary", "", "glossary name/FQN")
	glossaryTermCreateCmd.Flags().StringVar(&termParent, "parent", "", "parent term name/FQN")
	glossaryTermCreateCmd.Flags().StringVar(&termName, "name", "", "term name")
	glossaryTermCreateCmd.Flags().StringVar(&termDescription, "description", "", "term description")

	glossaryTermCmd.AddCommand(glossaryTermCreateCmd)
	glossaryCmd.AddCommand(glossaryListCmd)
	glossaryCmd.AddCommand(glossaryCreateCmd)
	glossaryCmd.AddCommand(glossaryTermCmd)
}
