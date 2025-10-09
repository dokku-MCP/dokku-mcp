package domain

// Planner DTOs for MCP tool responses (domain-visible types)

type PlanStep struct {
	Tool    string            `json:"tool"`
	Params  map[string]string `json:"params"`
	Why     string            `json:"why,omitempty"`
	DryRun  bool              `json:"dryRun,omitempty"`
	Confirm bool              `json:"confirm,omitempty"`
}

type Plan struct {
	Steps                []PlanStep `json:"steps"`
	SafetyChecklist      []string   `json:"safetyChecklist"`
	RequiresConfirmation bool       `json:"requiresConfirmation"`
}

type PlanResponse struct {
	Plan Plan `json:"plan"`
}
