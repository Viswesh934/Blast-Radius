package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	apiMethod      string
	apiPath        string
	apiBody        string
	apiBodyFile    string
	apiQueryParams []string
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Call OpenMetadata API directly",
	Long:  "Execute a raw OpenMetadata API call when you need functionality not yet wrapped by dedicated blast commands.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(apiPath) == "" {
			return fmt.Errorf("--path is required")
		}
		client, err := newOMClient()
		if err != nil {
			return err
		}

		payload, err := loadJSONPayload(apiBody, apiBodyFile)
		if err != nil {
			return err
		}
		query, err := parseQueryParams(apiQueryParams)
		if err != nil {
			return err
		}

		res, err := client.Call(
			context.Background(),
			strings.TrimSpace(apiMethod),
			strings.TrimSpace(apiPath),
			query,
			payload,
		)
		if err != nil {
			return err
		}

		if wantsJSONOutput() {
			return printStructured(map[string]any{"response": res})
		}

		keys := make([]string, 0, len(res))
		for k := range res {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := res[k]
			fmt.Printf("%s: %v\n", k, v)
		}
		if len(res) == 0 {
			fmt.Println("status: ok")
		}
		return nil
	},
}

func init() {
	apiCmd.Flags().StringVar(&apiMethod, "method", "GET", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	apiCmd.Flags().StringVar(&apiPath, "path", "", "OpenMetadata API path, e.g. /services/databaseServices")
	apiCmd.Flags().StringVar(&apiBody, "body", "", "raw JSON request body")
	apiCmd.Flags().StringVar(&apiBodyFile, "body-file", "", "path to JSON request body file")
	apiCmd.Flags().StringSliceVar(&apiQueryParams, "query", nil, "query parameter in key=value form; may be repeated")
}

func parseQueryParams(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, entry := range values {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid query param %q (expected key=value)", entry)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid query param %q (empty key)", entry)
		}
		out[key] = val
	}
	return out, nil
}
