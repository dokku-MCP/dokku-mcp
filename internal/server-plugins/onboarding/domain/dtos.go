package domain

import "time"

type PromptMeta struct {
	Plugin      string `json:"plugin"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PromptsCapabilities struct {
	Version     string       `json:"version"`
	GeneratedAt time.Time    `json:"generatedAt"`
	Prompts     []PromptMeta `json:"prompts"`
}

type IntentEntry struct {
	Synonyms []string `json:"synonyms"`
	Tool     string   `json:"tool"`
	Params   []string `json:"params"`
}

type IntentMap map[string]IntentEntry

type RecipeStep struct {
	Tool   string            `json:"tool"`
	Params map[string]string `json:"params"`
}

type Recipe struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	Preconditions  []string     `json:"preconditions"`
	Steps          []RecipeStep `json:"steps"`
	Postconditions []string     `json:"postconditions"`
}

type Examples struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Recipes     []Recipe  `json:"recipes"`
}

type CapabilityToolExampleParams struct {
	AppName      string `json:"app_name,omitempty"`
	RepoURL      string `json:"repo_url,omitempty"`
	GitRef       string `json:"git_ref,omitempty"`
	ValidateOnly bool   `json:"validateOnly,omitempty"`
}

type CapabilityToolExample struct {
	Tool   string                      `json:"tool"`
	Params CapabilityToolExampleParams `json:"params,omitempty"`
}

type CapabilityTool struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Examples    []CapabilityToolExample `json:"examples,omitempty"`
}

type CapabilityResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType"`
}

type CapabilityIndex struct {
	Tools     []CapabilityTool     `json:"tools"`
	Resources []CapabilityResource `json:"resources"`
	Prompts   []PromptMeta         `json:"prompts"`
}

func NewCapabilityIndex() CapabilityIndex {
	return CapabilityIndex{
		Tools:     make([]CapabilityTool, 0),
		Resources: make([]CapabilityResource, 0),
		Prompts:   make([]PromptMeta, 0),
	}
}
