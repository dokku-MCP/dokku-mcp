package auth

import (
	"context"
	"time"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
)

type NoOpAuthenticator struct{}

func NewNoOpAuthenticator() *NoOpAuthenticator {
	return &NoOpAuthenticator{}
}

func (a *NoOpAuthenticator) Authenticate(ctx context.Context, token string) (*shared.TenantContext, error) {
	return &shared.TenantContext{
		TenantID:        "default",
		UserID:          "default",
		Permissions:     []string{"*"},
		Metadata:        make(map[string]string),
		AuthenticatedAt: time.Now(),
	}, nil
}

type NoOpAuthorizationChecker struct{}

func NewNoOpAuthorizationChecker() *NoOpAuthorizationChecker {
	return &NoOpAuthorizationChecker{}
}

func (c *NoOpAuthorizationChecker) CheckPermission(ctx context.Context, tenant *shared.TenantContext, resource, action string) error {
	return nil
}

type NoOpSecretProvider struct{}

func NewNoOpSecretProvider() *NoOpSecretProvider {
	return &NoOpSecretProvider{}
}

func (p *NoOpSecretProvider) GetSSHConfig(ctx context.Context, tenantID string) (*SSHConfig, error) {
	return nil, nil
}

func (p *NoOpSecretProvider) GetSecret(ctx context.Context, tenantID, secretName string) (string, error) {
	return "", nil
}
