package workflows

// WorkflowStep defines a single action within a larger workflow.
type WorkflowStep struct {
	Name        string                 `yaml:"name" json:"name"`
	Type        string                 `yaml:"type" json:"type"` // e.g., "tool_call", "read_resource", "prompt"
	Target      string                 `yaml:"target" json:"target"`
	Arguments   map[string]interface{} `yaml:"arguments" json:"arguments"`
	Condition   string                 `yaml:"condition,omitempty" json:"condition,omitempty"`
	OutputAlias string                 `yaml:"output_alias,omitempty" json:"output_alias,omitempty"`
}

// WorkflowParameter represents a parameter definition for a workflow.
type WorkflowParameter struct {
	Name        string      `yaml:"name" json:"name"`
	Type        string      `yaml:"type" json:"type"`
	Description string      `yaml:"description" json:"description"`
	Required    bool        `yaml:"required" json:"required"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
}

// Workflow represents a declarative, multi-step operational playbook.
type Workflow struct {
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description" json:"description"`
	OwnerPlugin string              `yaml:"owner_plugin,omitempty" json:"owner_plugin,omitempty"`
	Arguments   []WorkflowParameter `yaml:"arguments" json:"arguments"`
	Steps       []WorkflowStep      `yaml:"steps" json:"steps"`
}

// WorkflowProvider defines the interface for loading workflow definitions.
type WorkflowProvider interface {
	// GetWorkflows returns all available workflow definitions.
	GetWorkflows() ([]Workflow, error)

	// GetWorkflow returns a specific workflow by name.
	GetWorkflow(name string) (*Workflow, error)
}

// WorkflowExecutionContext holds the state during workflow execution.
type WorkflowExecutionContext struct {
	Variables    map[string]interface{}
	CurrentStep  int
	WorkflowName string
}

// StepResult represents the result of executing a single workflow step.
type StepResult struct {
	StepName string      `json:"step_name"`
	Success  bool        `json:"success"`
	Output   interface{} `json:"output,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// WorkflowResult represents the result of executing an entire workflow.
type WorkflowResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	StepResults []StepResult           `json:"step_results"`
	Variables   map[string]interface{} `json:"variables"`
	Error       string                 `json:"error,omitempty"`
}
