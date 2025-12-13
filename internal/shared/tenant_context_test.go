package shared_test

import (
	"context"
	"time"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TenantContext", func() {
	var tenantCtx *shared.TenantContext

	BeforeEach(func() {
		tenantCtx = &shared.TenantContext{
			TenantID:        "tenant-123",
			UserID:          "user-456",
			Permissions:     []string{"apps:read", "apps:deploy"},
			Metadata:        map[string]string{"server_id": "server-789"},
			AuthenticatedAt: time.Now(),
		}
	})

	Describe("HasPermission", func() {
		It("returns true for existing permission", func() {
			Expect(tenantCtx.HasPermission("apps:read")).To(BeTrue())
			Expect(tenantCtx.HasPermission("apps:deploy")).To(BeTrue())
		})

		It("returns false for non-existing permission", func() {
			Expect(tenantCtx.HasPermission("apps:destroy")).To(BeFalse())
		})
	})

	Describe("IsExpired", func() {
		It("returns false when ExpiresAt is nil", func() {
			Expect(tenantCtx.IsExpired()).To(BeFalse())
		})

		It("returns false when not expired", func() {
			future := time.Now().Add(1 * time.Hour)
			tenantCtx.ExpiresAt = &future
			Expect(tenantCtx.IsExpired()).To(BeFalse())
		})

		It("returns true when expired", func() {
			past := time.Now().Add(-1 * time.Hour)
			tenantCtx.ExpiresAt = &past
			Expect(tenantCtx.IsExpired()).To(BeTrue())
		})
	})

	Describe("Metadata", func() {
		It("retrieves existing metadata", func() {
			value, exists := tenantCtx.GetMetadata("server_id")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("server-789"))
		})

		It("returns false for non-existing metadata", func() {
			_, exists := tenantCtx.GetMetadata("non_existing")
			Expect(exists).To(BeFalse())
		})

		It("sets new metadata", func() {
			tenantCtx.SetMetadata("new_key", "new_value")
			value, exists := tenantCtx.GetMetadata("new_key")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("new_value"))
		})

		It("initializes metadata map if nil", func() {
			emptyCtx := &shared.TenantContext{}
			emptyCtx.SetMetadata("key", "value")
			value, exists := emptyCtx.GetMetadata("key")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("value"))
		})
	})

	Describe("Context Integration", func() {
		It("stores and retrieves tenant context", func() {
			ctx := context.Background()
			ctx = shared.WithTenantContext(ctx, tenantCtx)

			retrieved, ok := shared.GetTenantContext(ctx)
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(tenantCtx))
		})

		It("returns false when tenant context not found", func() {
			ctx := context.Background()
			_, ok := shared.GetTenantContext(ctx)
			Expect(ok).To(BeFalse())
		})

		It("MustGetTenantContext returns context when present", func() {
			ctx := shared.WithTenantContext(context.Background(), tenantCtx)
			retrieved := shared.MustGetTenantContext(ctx)
			Expect(retrieved).To(Equal(tenantCtx))
		})

		It("MustGetTenantContext panics when context not found", func() {
			ctx := context.Background()
			Expect(func() {
				shared.MustGetTenantContext(ctx)
			}).To(Panic())
		})
	})
})
