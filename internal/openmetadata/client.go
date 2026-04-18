package openmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
	client.SetHeader("Accept", "application/json")
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

func (c *Client) apiError(action string, resp *resty.Response) error {
	var body struct {
		Code            int    `json:"code"`
		Message         string `json:"message"`
		ResponseMessage string `json:"responseMessage"`
	}
	_ = json.Unmarshal(resp.Body(), &body)

	message := strings.TrimSpace(body.Message)
	if message == "" {
		message = strings.TrimSpace(body.ResponseMessage)
	}
	if message != "" {
		if body.Code != 0 {
			return fmt.Errorf("%s failed: status=%d code=%d message=%s", action, resp.StatusCode(), body.Code, message)
		}
		return fmt.Errorf("%s failed: status=%d message=%s", action, resp.StatusCode(), message)
	}
	return fmt.Errorf("%s failed: status=%d body=%s", action, resp.StatusCode(), resp.String())
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
		return nil, c.apiError("list tables", resp)
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
		return nil, c.apiError("fetch lineage", resp)
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

func (c *Client) Call(ctx context.Context, method, path string, query map[string]string, body any) (map[string]any, error) {
	req := c.httpClient.R().SetContext(ctx)
	for k, v := range query {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		req.SetQueryParam(k, v)
	}
	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Execute(strings.ToUpper(strings.TrimSpace(method)), path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, c.apiError(fmt.Sprintf("openmetadata api request method=%s path=%s", strings.ToUpper(strings.TrimSpace(method)), path), resp)
	}

	out := map[string]any{}
	if len(resp.Body()) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(resp.Body(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListDatabaseServices(ctx context.Context) ([]DatabaseService, error) {
	resp, err := c.httpClient.R().SetContext(ctx).Get("/services/databaseServices")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("list database services", resp)
	}

	var result struct {
		Data []DatabaseService `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) CreateDatabaseService(ctx context.Context, payload map[string]any) (*DatabaseService, error) {
	resp, err := c.httpClient.R().SetContext(ctx).SetBody(payload).Post("/services/databaseServices")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated && resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("create database service", resp)
	}

	out := &DatabaseService{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetDatabaseServiceByName(ctx context.Context, serviceName string) (*EntityReference, error) {
	escaped := url.PathEscape(serviceName)
	resp, err := c.httpClient.R().SetContext(ctx).Get("/services/databaseServices/name/" + escaped)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("lookup database service name="+serviceName, resp)
	}

	var svc DatabaseService
	if err := json.Unmarshal(resp.Body(), &svc); err != nil {
		return nil, err
	}
	return &EntityReference{ID: svc.ID, Name: svc.Name, Type: "databaseService", FullyQualifiedName: svc.FullyQualifiedName}, nil
}

func (c *Client) ListDatabases(ctx context.Context, serviceName string) ([]Database, error) {
	req := c.httpClient.R().SetContext(ctx)
	if strings.TrimSpace(serviceName) != "" {
		req.SetQueryParam("service", strings.TrimSpace(serviceName))
	}
	resp, err := req.Get("/databases")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("list databases", resp)
	}

	var result struct {
		Data []Database `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) CreateDatabase(ctx context.Context, name, serviceName, description string) (*Database, error) {
	serviceRef, err := c.GetDatabaseServiceByName(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"name":    name,
		"service": serviceRef.FullyQualifiedName,
	}
	if strings.TrimSpace(description) != "" {
		payload["description"] = description
	}

	resp, err := c.httpClient.R().SetContext(ctx).SetBody(payload).Post("/databases")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated && resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("create database", resp)
	}

	out := &Database{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetDatabaseByName(ctx context.Context, databaseFQN string) (*EntityReference, error) {
	escaped := url.PathEscape(databaseFQN)
	resp, err := c.httpClient.R().SetContext(ctx).Get("/databases/name/" + escaped)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("lookup database name="+databaseFQN, resp)
	}

	var db Database
	if err := json.Unmarshal(resp.Body(), &db); err != nil {
		return nil, err
	}
	return &EntityReference{ID: db.ID, Name: db.Name, Type: "database", FullyQualifiedName: db.FullyQualifiedName}, nil
}

func (c *Client) CreateDatabaseSchema(ctx context.Context, name, databaseFQN, description string) (*DatabaseSchema, error) {
	dbRef, err := c.GetDatabaseByName(ctx, databaseFQN)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"name":     name,
		"database": dbRef.FullyQualifiedName,
	}
	if strings.TrimSpace(description) != "" {
		payload["description"] = description
	}

	resp, err := c.httpClient.R().SetContext(ctx).SetBody(payload).Post("/databaseSchemas")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated && resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("create database schema", resp)
	}

	out := &DatabaseSchema{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetDatabaseSchemaByName(ctx context.Context, schemaFQN string) (*EntityReference, error) {
	escaped := url.PathEscape(schemaFQN)
	resp, err := c.httpClient.R().SetContext(ctx).Get("/databaseSchemas/name/" + escaped)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("lookup database schema name="+schemaFQN, resp)
	}

	var schema DatabaseSchema
	if err := json.Unmarshal(resp.Body(), &schema); err != nil {
		return nil, err
	}
	return &EntityReference{ID: schema.ID, Name: schema.Name, Type: "databaseSchema", FullyQualifiedName: schema.FullyQualifiedName}, nil
}

func (c *Client) GetTableByName(ctx context.Context, tableFQN string) (*TableEntity, error) {
	escaped := url.PathEscape(tableFQN)
	resp, err := c.httpClient.R().SetContext(ctx).Get("/tables/name/" + escaped)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("lookup table name="+tableFQN, resp)
	}

	out := &TableEntity{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteTable(ctx context.Context, tableFQN string) error {
	table, err := c.GetTableByName(ctx, tableFQN)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.R().SetContext(ctx).Delete("/tables/" + url.PathEscape(table.ID))
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return c.apiError("delete table", resp)
	}
	return nil
}

func (c *Client) UpdateTable(ctx context.Context, tableFQN, method string, payload any) (*TableEntity, error) {
	table, err := c.GetTableByName(ctx, tableFQN)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(method) == "" {
		method = http.MethodPatch
	}
	req := c.httpClient.R().SetContext(ctx).SetBody(payload)
	if strings.EqualFold(strings.TrimSpace(method), http.MethodPatch) {
		switch payload.(type) {
		case []any:
			req.SetHeader("Content-Type", "application/json-patch+json")
		}
	}

	resp, err := req.Execute(strings.ToUpper(strings.TrimSpace(method)), "/tables/"+url.PathEscape(table.ID))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, c.apiError("update table", resp)
	}

	out := &TableEntity{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		out.ID = table.ID
	}
	if out.FullyQualifiedName == "" {
		out.FullyQualifiedName = table.FullyQualifiedName
	}
	if out.Name == "" {
		out.Name = table.Name
	}
	return out, nil
}

func (c *Client) CreateTable(ctx context.Context, name, schemaFQN, description string, payload map[string]any) (*TableEntity, error) {
	if payload == nil {
		schemaRef, err := c.GetDatabaseSchemaByName(ctx, schemaFQN)
		if err != nil {
			return nil, err
		}
		payload = map[string]any{
			"name":           strings.TrimSpace(name),
			"databaseSchema": schemaRef.FullyQualifiedName,
		}
		if strings.TrimSpace(description) != "" {
			payload["description"] = strings.TrimSpace(description)
		}
	}

	resp, err := c.httpClient.R().SetContext(ctx).SetBody(payload).Post("/tables")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated && resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("create table", resp)
	}

	out := &TableEntity{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListGlossaries(ctx context.Context) ([]Glossary, error) {
	resp, err := c.httpClient.R().SetContext(ctx).Get("/glossaries")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("list glossaries", resp)
	}

	var result struct {
		Data []Glossary `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) CreateGlossary(ctx context.Context, name, description string) (*Glossary, error) {
	payload := map[string]any{"name": strings.TrimSpace(name)}
	if strings.TrimSpace(description) != "" {
		payload["description"] = strings.TrimSpace(description)
	}

	resp, err := c.httpClient.R().SetContext(ctx).SetBody(payload).Post("/glossaries")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated && resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("create glossary", resp)
	}

	out := &Glossary{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListGlossaryTerms(ctx context.Context, glossaryName string) ([]GlossaryTerm, error) {
	path := "/glossaryTerms"
	if strings.TrimSpace(glossaryName) != "" {
		path = "/glossaries/name/" + url.PathEscape(strings.TrimSpace(glossaryName)) + "/terms"
	}

	resp, err := c.httpClient.R().SetContext(ctx).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("list glossary terms", resp)
	}

	var result struct {
		Data []GlossaryTerm `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func (c *Client) CreateGlossaryTerm(ctx context.Context, glossaryName, parentName, name, description string) (*GlossaryTerm, error) {
	payload := map[string]any{"name": strings.TrimSpace(name)}
	if strings.TrimSpace(description) != "" {
		payload["description"] = strings.TrimSpace(description)
	}
	if strings.TrimSpace(glossaryName) != "" {
		payload["glossary"] = map[string]any{"name": strings.TrimSpace(glossaryName)}
	}
	if strings.TrimSpace(parentName) != "" {
		payload["parent"] = map[string]any{"name": strings.TrimSpace(parentName)}
	}

	resp, err := c.httpClient.R().SetContext(ctx).SetBody(payload).Post("/glossaryTerms")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated && resp.StatusCode() != http.StatusOK {
		return nil, c.apiError("create glossary term", resp)
	}

	out := &GlossaryTerm{}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return nil, err
	}
	return out, nil
}
