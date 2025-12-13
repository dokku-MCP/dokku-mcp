package auth

import (
	"context"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*shared.TenantContext, error)
}

type AuthorizationChecker interface {
	CheckPermission(ctx context.Context, tenant *shared.TenantContext, resource, action string) error
}

type SecretProvider interface {
	GetSSHConfig(ctx context.Context, tenantID string) (*SSHConfig, error)
	GetSecret(ctx context.Context, tenantID, secretName string) (string, error)
}

type SSHConfig struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	KeyPath    string
}
