package openmetadata

type Table struct {
	FQN         string   `json:"fullyQualifiedName"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	UpdatedAt   int64    `json:"updatedAt"`
	Columns     []Column `json:"columns"`
}

type Column struct {
	Name        string `json:"name"`
	DataType    string `json:"dataType"`
	Description string `json:"description"`
	Nullable    bool   `json:"nullable"`
	Constraint  string `json:"constraint"`
}

type LineageData struct {
	UpstreamFQNs   []string `json:"upstream_fqns"`
	DownstreamFQNs []string `json:"downstream_fqns"`
}
