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

type EntityReference struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Name               string `json:"name"`
	FullyQualifiedName string `json:"fullyQualifiedName"`
	DisplayName        string `json:"displayName"`
	Deleted            bool   `json:"deleted"`
}

type DatabaseService struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	FullyQualifiedName string `json:"fullyQualifiedName"`
	ServiceType        string `json:"serviceType"`
	Description        string `json:"description"`
}

type Database struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	FullyQualifiedName string          `json:"fullyQualifiedName"`
	Description        string          `json:"description"`
	Service            EntityReference `json:"service"`
}

type DatabaseSchema struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	FullyQualifiedName string          `json:"fullyQualifiedName"`
	Description        string          `json:"description"`
	Database           EntityReference `json:"database"`
}

type TableEntity struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	FullyQualifiedName string          `json:"fullyQualifiedName"`
	Description        string          `json:"description"`
	DatabaseSchema     EntityReference `json:"databaseSchema"`
}

type Glossary struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	FullyQualifiedName string `json:"fullyQualifiedName"`
	Description        string `json:"description"`
}

type GlossaryTerm struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	FullyQualifiedName string          `json:"fullyQualifiedName"`
	Description        string          `json:"description"`
	Glossary           EntityReference `json:"glossary"`
	Parent             EntityReference `json:"parent"`
}
