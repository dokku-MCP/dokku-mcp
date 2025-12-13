package authorization_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/authorization"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuthorizedTool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AuthorizedTool Suite")
}

// Mock authorization checker
type mockAuthChecker struct {
	shouldAllow bool
	checkCalled bool
}

func (m *mockAuthChecker) CheckPermission(ctx context.Context, tenant *shared.TenantContext, resource, action string) error {
	m.checkCalled = true
	if !m.shouldAllow {
		return errors.New("permission denied")
	}
	return nil
}

var _ = Describe("WrapToolWithAuthorization", func() {
	var (
		baseTool    domain.Tool
		authChecker *mockAuthChecker
		logger      *slog.Logger
		ctx         context.Context
		executed    bool
	)

	BeforeEach(func() {
		executed = false
		baseTool = domain.Tool{
			Name:        "test-tool",
			Description: "Test tool",
			Builder:     nil, // Not needed for this test
			Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				executed = true
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{Type: "text", Text: "success"},
					},
				}, nil
			},
		}
		authChecker = &mockAuthChecker{shouldAllow: true}
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		ctx = context.Background()
	})

	Describe("Without tenant context", func() {
		It("executes tool without authorization check", func() {
			wrappedTool := authorization.WrapToolWithAuthorization(
				baseTool,
				"apps",
				"deploy",
				authChecker,
				logger,
			)

			result, err := wrappedTool.Handler(ctx, mcp.CallToolRequest{})

			Expect(err).ToNot(HaveOccurred())
			Expect(executed).To(BeTrue())
			Expect(authChecker.checkCalled).To(BeFalse())
			_ = result
		})
	})

	Describe("With tenant context", func() {
		var tenantCtx *shared.TenantContext

		BeforeEach(func() {
			tenantCtx = &shared.TenantContext{
				TenantID:    "tenant-123",
				UserID:      "user-456",
				Permissions: []string{"apps:deploy"},
			}
			ctx = shared.WithTenantContext(ctx, tenantCtx)
		})

		It("performs authorization check and executes on success", func() {
			authChecker.shouldAllow = true

			wrappedTool := authorization.WrapToolWithAuthorization(
				baseTool,
				"apps",
				"deploy",
				authChecker,
				logger,
			)

			result, err := wrappedTool.Handler(ctx, mcp.CallToolRequest{})

			Expect(err).ToNot(HaveOccurred())
			Expect(executed).To(BeTrue())
			Expect(authChecker.checkCalled).To(BeTrue())
			_ = result
		})

		It("returns error when authorization fails", func() {
			authChecker.shouldAllow = false

			wrappedTool := authorization.WrapToolWithAuthorization(
				baseTool,
				"apps",
				"destroy",
				authChecker,
				logger,
			)

			result, err := wrappedTool.Handler(ctx, mcp.CallToolRequest{})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("permission denied"))
			Expect(executed).To(BeFalse())
			Expect(authChecker.checkCalled).To(BeTrue())
			_ = result
		})
	})

	Describe("Tool properties", func() {
		It("preserves tool name and description", func() {
			wrappedTool := authorization.WrapToolWithAuthorization(
				baseTool,
				"apps",
				"deploy",
				authChecker,
				logger,
			)

			Expect(wrappedTool.Name).To(Equal("test-tool"))
			Expect(wrappedTool.Description).To(Equal("Test tool"))
		})
	})
})
