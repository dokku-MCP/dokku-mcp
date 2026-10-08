// Package plugintest helps test server plugins through their MCP tools.
package plugintest

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

// Args are the arguments of a tool call.
type Args = map[string]any // NOTE: MCP tool arguments are untyped JSON objects. This is a valid exception

// FindTool returns the named tool of a tool provider, failing the test if
// it does not exist.
func FindTool(t *testing.T, provider domain.ToolProvider, name string) domain.Tool {
	t.Helper()
	tools, err := provider.GetTools(context.Background())
	if err != nil {
		t.Fatalf("GetTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return domain.Tool{}
}

// CallTool invokes a tool handler with the given arguments.
func CallTool(t *testing.T, provider domain.ToolProvider, name string, args Args) *mcp.CallToolResult {
	t.Helper()
	tool := FindTool(t, provider, name)
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("%s returned a protocol error: %v", name, err)
	}
	if result == nil {
		t.Fatalf("%s returned a nil result", name)
	}
	return result
}

// Text concatenates the text content of a tool result.
func Text(result *mcp.CallToolResult) string {
	var parts []string
	for _, c := range result.Content {
		if text, ok := c.(mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// RequireSuccess fails the test when the tool reported an error.
func RequireSuccess(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()
	if result.IsError {
		t.Fatalf("expected success, got error: %s", Text(result))
	}
}

// RequireError fails the test unless the tool reported an error containing
// the given substring.
func RequireError(t *testing.T, result *mcp.CallToolResult, contains string) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("expected an error, got: %s", Text(result))
	}
	if !strings.Contains(Text(result), contains) {
		t.Fatalf("expected error containing %q, got: %s", contains, Text(result))
	}
}

// Structured decodes the structured content of a successful tool result.
func Structured[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	RequireSuccess(t, result)
	if result.StructuredContent == nil {
		t.Fatalf("expected structured content, got text-only result: %s", Text(result))
	}
	var out T
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	return out
}

// RequireAnnotated fails the test unless every tool of the provider has a
// title annotation and consistent hints. mcp-go defaults every tool to
// destructive and open-world, so a tool without a title most likely never
// had its hints reviewed.
func RequireAnnotated(t *testing.T, provider domain.ToolProvider) {
	t.Helper()
	tools, err := provider.GetTools(context.Background())
	if err != nil {
		t.Fatalf("GetTools: %v", err)
	}
	for _, tool := range tools {
		ann := tool.Builder().Annotations
		if ann.Title == "" {
			t.Errorf("tool %q must set a title annotation", tool.Name)
		}
		if ann.ReadOnlyHint != nil && *ann.ReadOnlyHint && (ann.DestructiveHint == nil || *ann.DestructiveHint) {
			t.Errorf("read-only tool %q must not be marked destructive", tool.Name)
		}
		if tool.Builder().Name != tool.Name {
			t.Errorf("tool %q builds an MCP tool named %q", tool.Name, tool.Builder().Name)
		}
	}
}
