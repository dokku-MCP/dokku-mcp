package dokku

import (
	"context"
	"errors"
	"testing"

	dokku_client "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/domain"
)

type fakeClient struct{}

func (f *fakeClient) ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	return nil, &dokku_client.NotFoundError{Command: command, Err: errors.New("App does not exist")}
}

// satisfy interfaces used by status checker but not needed for this test
func (f *fakeClient) GetKeyValueOutput(ctx context.Context, command string, args []string, separator string) (map[string]string, error) {
	return nil, nil
}
func (f *fakeClient) GetListOutput(ctx context.Context, command string, args []string) ([]string, error) {
	return nil, nil
}
func (f *fakeClient) GetTableOutput(ctx context.Context, command string, args []string, skipHeaders bool) ([]map[string]string, error) {
	return nil, nil
}
func (f *fakeClient) ExecuteStructured(ctx context.Context, spec dokku_client.CommandSpec) (*dokku_client.CommandResult, error) {
	return nil, nil
}
func (f *fakeClient) ExecuteWithAutoFormat(ctx context.Context, commandName string, args []string) (*dokku_client.CommandResult, error) {
	return nil, nil
}
func (f *fakeClient) DiscoverCapabilities(ctx context.Context) error { return nil }
func (f *fakeClient) GetCapabilities() *dokku_client.DokkuCapabilities {
	return dokku_client.NewDokkuCapabilities()
}
func (f *fakeClient) GetSSHConnectionManager() *dokku_client.SSHConnectionManager { return nil }
func (f *fakeClient) SetBlacklist(commands []string)                              {}
func (f *fakeClient) ValidateCommand(command string, args []string) error         { return nil }

func TestStatusCheckerNotFoundReturnsFailed(t *testing.T) {
	dsc := NewDeploymentStatusChecker(&fakeClient{})
	status, msg, err := dsc.CheckStatus(context.Background(), "ghost-app")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != domain.DeploymentStatusFailed {
		t.Fatalf("expected Failed status, got %v", status)
	}
	if msg == "" {
		t.Fatalf("expected a non-empty message")
	}
}
