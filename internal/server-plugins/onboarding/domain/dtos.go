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
