package app

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestFollowRuntimeLogsStreamsProgress runs follow_runtime_logs through a
// real MCP server and client to check that lines arrive as progress
// notifications before the result.
func TestFollowRuntimeLogsStreamsProgress(t *testing.T) {
	f := newFixture(t)
	f.client.Respond("logs", "booting\nlistening on :5000\n")

	srv := server.NewMCPServer("test", "0.0.0", server.WithToolCapabilities(true))
	tools, err := f.apps.GetTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		srv.AddTool(tool.Builder(), tool.Handler)
	}

	c, err := client.NewInProcessClient(srv)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var messages []string
	c.OnNotification(func(n mcp.JSONRPCNotification) {
		if n.Method != string(mcp.MethodNotificationProgress) {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if msg, ok := n.Params.AdditionalFields["message"].(string); ok {
			messages = append(messages, msg)
		}
	})

	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "0"}
	if _, err := c.Initialize(ctx, init); err != nil {
		t.Fatal(err)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "follow_runtime_logs"
	req.Params.Arguments = map[string]any{"app_name": "myapp", "seconds": 5}
	req.Params.Meta = &mcp.Meta{ProgressToken: "follow-1"}
	result, err := c.CallTool(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool failed: %+v", result.Content)
	}

	// The in-process client dispatches notifications asynchronously, so
	// they may still be in flight when the result arrives.
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		got := slices.Clone(messages)
		mu.Unlock()
		if slices.Equal(got, []string{"booting", "listening on :5000"}) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("progress messages = %q", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
