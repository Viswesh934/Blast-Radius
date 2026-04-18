package openmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Client struct {
	baseURL    string
	jwtToken   string
	httpClient *resty.Client
	logger     *zap.Logger
}

func NewClient(baseURL, jwtToken string, logger *zap.Logger) *Client {
	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetHeader("Content-Type", "application/json")
	if jwtToken != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	}

	return &Client{
		baseURL:    baseURL,
		jwtToken:   jwtToken,
		httpClient: client,
		logger:     logger,
	}
}

func (c *Client) GetTablesByDatabase(ctx context.Context, databaseFQN string) ([]Table, error) {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("database", databaseFQN).
		Get("/tables")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("tables request failed: status=%d body=%s", resp.StatusCode(), resp.String())
	}

	var result struct {
		Data []Table `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) GetLineage(ctx context.Context, entityFQN string, upstreamDepth, downstreamDepth int) (*LineageData, error) {
	escapedFQN := url.PathEscape(entityFQN)
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("upstreamDepth", fmt.Sprintf("%d", upstreamDepth)).
		SetQueryParam("downstreamDepth", fmt.Sprintf("%d", downstreamDepth)).
		Get("/lineage/table/name/" + escapedFQN)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("lineage request failed: status=%d", resp.StatusCode())
	}

	var raw struct {
		Nodes []struct {
			FullyQualifiedName string `json:"fullyQualifiedName"`
			Id                 string `json:"id"`
		} `json:"nodes"`
		UpstreamEdges []struct {
			FromEntity string `json:"fromEntity"`
			ToEntity   string `json:"toEntity"`
		} `json:"upstreamEdges"`
		DownstreamEdges []struct {
			FromEntity string `json:"fromEntity"`
			ToEntity   string `json:"toEntity"`
		} `json:"downstreamEdges"`
	}
	if err := json.Unmarshal(resp.Body(), &raw); err != nil {
		return nil, err
	}

	nodeByID := make(map[string]string, len(raw.Nodes))
	for _, n := range raw.Nodes {
		nodeByID[n.Id] = n.FullyQualifiedName
	}

	lineage := &LineageData{
		UpstreamFQNs:   make([]string, 0),
		DownstreamFQNs: make([]string, 0),
	}
	for _, edge := range raw.UpstreamEdges {
		if fqn, ok := nodeByID[edge.FromEntity]; ok {
			lineage.UpstreamFQNs = append(lineage.UpstreamFQNs, fqn)
		}
	}
	for _, edge := range raw.DownstreamEdges {
		if fqn, ok := nodeByID[edge.ToEntity]; ok {
			lineage.DownstreamFQNs = append(lineage.DownstreamFQNs, fqn)
		}
	}

	return lineage, nil
}
