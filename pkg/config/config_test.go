package config

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func loadWithEnv(t *testing.T, env map[string]string) *ServerConfig {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Chdir(t.TempDir()) // no config.yaml in the working directory
	t.Setenv("HOME", t.TempDir())
	// Clear DOKKU_MCP_* variables inherited from the shell or CI.
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "DOKKU_MCP_") {
			t.Setenv(name, "")
			_ = os.Unsetenv(name)
		}
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg
}

func TestLoadConfigRejectsInvalidAllowlistPattern(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Chdir(t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DOKKU_MCP_SECURITY_ALLOWLIST", "apps:,config set")
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected an allowlist error, got %v", err)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg := loadWithEnv(t, nil)
	if cfg.Transport.Type != "stdio" {
		t.Errorf("transport.type = %q, want stdio", cfg.Transport.Type)
	}
	if len(cfg.Security.Allowlist) != 0 {
		t.Errorf("allowlist should default to empty, got %v", cfg.Security.Allowlist)
	}
}

func TestLoadConfigNestedEnvironmentVariables(t *testing.T) {
	cfg := loadWithEnv(t, map[string]string{
		"DOKKU_MCP_SSH_HOST":           "dokku.example.com",
		"DOKKU_MCP_SSH_PORT":           "2222",
		"DOKKU_MCP_SSH_USER":           "deployer",
		"DOKKU_MCP_TRANSPORT_TYPE":     "sse",
		"DOKKU_MCP_SECURITY_BLACKLIST": "destroy,uninstall",
		"DOKKU_MCP_SECURITY_ALLOWLIST": "apps:,config:",
		"DOKKU_MCP_LOG_LEVEL":          "warn",
	})

	if cfg.SSH.Host != "dokku.example.com" || cfg.SSH.Port != 2222 || cfg.SSH.User != "deployer" {
		t.Errorf("ssh config not read from environment: %+v", cfg.SSH)
	}
	if cfg.Transport.Type != "sse" {
		t.Errorf("transport.type = %q, want sse", cfg.Transport.Type)
	}
	if !slices.Equal(cfg.Security.Blacklist, []string{"destroy", "uninstall"}) {
		t.Errorf("blacklist = %v", cfg.Security.Blacklist)
	}
	if !slices.Equal(cfg.Security.Allowlist, []string{"apps:", "config:"}) {
		t.Errorf("allowlist = %v", cfg.Security.Allowlist)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("log_level = %q, want warn", cfg.LogLevel)
	}
}
