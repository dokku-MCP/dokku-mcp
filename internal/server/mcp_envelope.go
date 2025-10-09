package server

import (
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolStatus represents the high-level status of a tool call
type ToolStatus string

const (
	ToolStatusOK      ToolStatus = "ok"
	ToolStatusError   ToolStatus = "error"
	ToolStatusPartial ToolStatus = "partial"
)

// ToolLink provides follow-up actions for LLMs to chain safely
type ToolLink struct {
	Rel    string         `json:"rel"`
	Tool   string         `json:"tool"`
	Params map[string]any `json:"params,omitempty"`
}

// ToolResponse is the canonical envelope returned by tools
type ToolResponse struct {
	Status    ToolStatus `json:"status"`
	Code      string     `json:"code,omitempty"`
	Message   string     `json:"message,omitempty"`
	RequestID string     `json:"requestId,omitempty"`
	Data      any        `json:"data,omitempty"`
	Links     []ToolLink `json:"links,omitempty"`
	Hint      string     `json:"hint,omitempty"`
}

// marshal pretty JSON for readability in clients
func (r ToolResponse) marshal(logger *slog.Logger) string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		if logger != nil {
			logger.Error("failed to marshal tool response", "error", err, "code", r.Code)
		}
		fallback := ToolResponse{Status: ToolStatusError, Code: "tool_response_marshal_error", Message: "failed to serialize tool response"}
		fb, _ := json.MarshalIndent(fallback, "", "  ")
		return string(fb)
	}
	return string(b)
}

// NewResult builds an MCP result from a ToolResponse in one line
func NewResult(resp ToolResponse) *mcp.CallToolResult {
	return NewResultWithLogger(resp, nil)
}

// NewResultWithLogger builds an MCP result from a ToolResponse using the provided logger
func NewResultWithLogger(resp ToolResponse, logger *slog.Logger) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resp.marshal(logger)}}}
}

// OK is a convenience for success responses
func OK(message string, data any) *mcp.CallToolResult {
	return NewResult(ToolResponse{Status: ToolStatusOK, Message: message, Data: data})
}

// Error is a convenience for error responses
func Error(code, message, hint string, data any) *mcp.CallToolResult {
	return NewResult(ToolResponse{Status: ToolStatusError, Code: code, Message: message, Hint: hint, Data: data})
}

// Partial is a convenience for partial success responses
func Partial(message string, data any) *mcp.CallToolResult {
	return NewResult(ToolResponse{Status: ToolStatusPartial, Message: message, Data: data})
}
