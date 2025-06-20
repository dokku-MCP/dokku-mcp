//go:build integration

package dokkuApi_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSH Integration", func() {
	var (
		logger *slog.Logger
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(GinkgoWriter, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	})

	Describe("SSHConfig", func() {
		Describe("NewSSHConfigFromServerConfig", func() {
			It("should create SSH config from server config parameters", func() {
				config, err := dokkuApi.NewSSHConfigFromServerConfig(
					"example.com",
					2222,
					"testuser",
					"/path/to/key",
					45*time.Second,
				)

				Expect(err).NotTo(HaveOccurred())
				Expect(config.Host()).To(Equal("example.com"))
				Expect(config.Port()).To(Equal(2222))
				Expect(config.User()).To(Equal("testuser"))
				Expect(config.KeyPath()).To(Equal("/path/to/key"))
				Expect(config.Timeout()).To(Equal(45 * time.Second))
			})

			It("should validate configuration parameters", func() {
				_, err := dokkuApi.NewSSHConfigFromServerConfig(
					"", // Invalid empty host
					22,
					"testuser",
					"",
					30*time.Second,
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("SSH host cannot be empty"))
			})
		})

		Describe("BaseSSHArgs", func() {
			It("should return correct base SSH arguments", func() {
				config := dokkuApi.MustNewSSHConfig("example.com", 2222, "testuser", "", 30*time.Second)
				args := config.BaseSSHArgs()

				Expect(args).To(ContainElement("-o"))
				Expect(args).To(ContainElement("LogLevel=QUIET"))
				Expect(args).To(ContainElement("StrictHostKeyChecking=no"))
				Expect(args).To(ContainElement("-p"))
				Expect(args).To(ContainElement("2222"))
			})
		})
	})

	Describe("SSHConnectionManager", func() {
		var (
			sshConfig *dokkuApi.SSHConfig
			manager   *dokkuApi.SSHConnectionManager
		)

		BeforeEach(func() {
			sshConfig = dokkuApi.MustNewSSHConfig("example.com", 22, "testuser", "", 30*time.Second)
			manager = dokkuApi.NewSSHConnectionManager(sshConfig, logger)
		})

		Describe("NewSSHConnectionManagerFromServerConfig", func() {
			It("should create manager from server config", func() {
				serverConfig := &config.ServerConfig{
					SSH: config.SSHConfig{
						Host:    "dokku.example.com",
						Port:    22,
						User:    "dokku",
						KeyPath: "/path/to/key",
					},
					Timeout: 60 * time.Second,
				}

				manager, err := dokkuApi.NewSSHConnectionManagerFromServerConfig(serverConfig, logger)

				Expect(err).NotTo(HaveOccurred())
				Expect(manager.Config().Host()).To(Equal("dokku.example.com"))
				Expect(manager.Config().User()).To(Equal("dokku"))
				Expect(manager.Config().KeyPath()).To(Equal("/path/to/key"))
			})
		})

		Describe("PrepareSSHCommand", func() {
			It("should prepare SSH command with authentication", func() {
				sshArgs, env, err := manager.PrepareSSHCommand("dokku apps:list")

				Expect(err).NotTo(HaveOccurred())
				Expect(sshArgs).To(ContainElement("ssh"))
				Expect(sshArgs).To(ContainElement("testuser@example.com"))
				Expect(sshArgs).To(ContainElement("dokku apps:list"))
				Expect(env).To(ContainElement("PATH=/usr/bin:/bin"))
			})

			It("should handle empty commands", func() {
				sshArgs, env, err := manager.PrepareSSHCommand("")

				Expect(err).NotTo(HaveOccurred())
				Expect(sshArgs).To(ContainElement("ssh"))
				Expect(sshArgs).To(ContainElement("testuser@example.com"))
				Expect(sshArgs).NotTo(ContainElement("--"))
				Expect(env).NotTo(BeEmpty())
			})
		})

		Describe("GetConnectionInfo", func() {
			It("should return connection information", func() {
				info := manager.GetConnectionInfo()

				Expect(info).To(HaveKey("host"))
				Expect(info).To(HaveKey("port"))
				Expect(info).To(HaveKey("user"))
				Expect(info).To(HaveKey("auth_method"))
				Expect(info).To(HaveKey("connection_string"))
				Expect(info["host"]).To(Equal("example.com"))
				Expect(info["connection_string"]).To(Equal("testuser@example.com"))
			})
		})
	})

	Describe("SSHConfigBuilder", func() {
		var builder *dokkuApi.SSHConfigBuilder

		BeforeEach(func() {
			builder = dokkuApi.NewSSHConfigBuilder(logger)
		})

		Describe("Fluent Interface", func() {
			It("should allow fluent configuration", func() {
				config, err := builder.
					WithHost("fluent.example.com").
					WithPort(2222).
					WithUser("fluentuser").
					WithKeyPath("/fluent/key").
					WithTimeout(45 * time.Second).
					Build()

				Expect(err).NotTo(HaveOccurred())
				Expect(config.Host()).To(Equal("fluent.example.com"))
				Expect(config.Port()).To(Equal(2222))
				Expect(config.User()).To(Equal("fluentuser"))
				Expect(config.KeyPath()).To(Equal("/fluent/key"))
				Expect(config.Timeout()).To(Equal(45 * time.Second))
			})
		})

		Describe("FromServerConfig", func() {
			It("should build from server configuration", func() {
				serverConfig := &config.ServerConfig{
					SSH: config.SSHConfig{
						Host:    "server.example.com",
						Port:    2222,
						User:    "serveruser",
						KeyPath: "/server/key",
					},
					Timeout: 60 * time.Second,
				}

				config, err := builder.FromServerConfig(serverConfig).Build()

				Expect(err).NotTo(HaveOccurred())
				Expect(config.Host()).To(Equal("server.example.com"))
				Expect(config.Port()).To(Equal(2222))
				Expect(config.User()).To(Equal("serveruser"))
				Expect(config.KeyPath()).To(Equal("/server/key"))
				Expect(config.Timeout()).To(Equal(60 * time.Second))
			})
		})

		Describe("BuildConnectionManager", func() {
			It("should build a complete connection manager", func() {
				manager, err := builder.
					WithHost("manager.example.com").
					WithUser("manageruser").
					BuildConnectionManager()

				Expect(err).NotTo(HaveOccurred())
				Expect(manager.Config().Host()).To(Equal("manager.example.com"))
				Expect(manager.Config().User()).To(Equal("manageruser"))
			})
		})
	})

	Describe("Integration with Authentication", func() {
		Context("when ssh-agent is available", func() {
			var (
				manager *dokkuApi.SSHConnectionManager
			)

			BeforeEach(func() {
				config := dokkuApi.MustNewSSHConfig("test.example.com", 22, "testuser", "", 30*time.Second)
				manager = dokkuApi.NewSSHConnectionManager(config, logger)
			})

			It("should prepare commands with ssh-agent authentication", func() {
				sshArgs, env, err := manager.PrepareSSHCommand("test command")

				Expect(err).NotTo(HaveOccurred())
				Expect(sshArgs).To(ContainElement("ssh"))
				Expect(env).To(ContainElement("PATH=/usr/bin:/bin"))
			})
		})

		Context("with custom SSH key", func() {
			var (
				manager     *dokkuApi.SSHConnectionManager
				tempKeyPath string
			)

			BeforeEach(func() {
				// Create a temporary key file
				tempFile, err := os.CreateTemp("", "test-ssh-key-*")
				Expect(err).NotTo(HaveOccurred())
				tempKeyPath = tempFile.Name()
				_, err = tempFile.WriteString("fake-ssh-key-content")
				Expect(err).NotTo(HaveOccurred())
				tempFile.Close()

				config := dokkuApi.MustNewSSHConfig("test.example.com", 22, "testuser", tempKeyPath, 30*time.Second)
				manager = dokkuApi.NewSSHConnectionManager(config, logger)

				DeferCleanup(func() {
					os.Remove(tempKeyPath)
				})
			})

			It("should include key path in SSH arguments", func() {
				sshArgs, _, err := manager.PrepareSSHCommand("test command")

				Expect(err).NotTo(HaveOccurred())
				Expect(sshArgs).To(ContainElement("-i"))
				keyIndex := -1
				for i, arg := range sshArgs {
					if arg == "-i" && i+1 < len(sshArgs) {
						keyIndex = i + 1
						break
					}
				}
				Expect(keyIndex).To(BeNumerically(">=", 0))
				Expect(sshArgs[keyIndex]).To(Equal(tempKeyPath))
			})
		})
	})
})

// Unit tests for specific validation scenarios
func TestSSHConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		port        int
		user        string
		keyPath     string
		timeout     time.Duration
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid configuration",
			host:        "valid.example.com",
			port:        22,
			user:        "validuser",
			keyPath:     "/valid/key",
			timeout:     30 * time.Second,
			expectError: false,
		},
		{
			name:        "empty host",
			host:        "",
			port:        22,
			user:        "validuser",
			keyPath:     "",
			timeout:     30 * time.Second,
			expectError: true,
			errorMsg:    "SSH host cannot be empty",
		},
		{
			name:        "invalid port - too low",
			host:        "valid.example.com",
			port:        0,
			user:        "validuser",
			keyPath:     "",
			timeout:     30 * time.Second,
			expectError: true,
			errorMsg:    "invalid SSH port",
		},
		{
			name:        "invalid port - too high",
			host:        "valid.example.com",
			port:        70000,
			user:        "validuser",
			keyPath:     "",
			timeout:     30 * time.Second,
			expectError: true,
			errorMsg:    "invalid SSH port",
		},
		{
			name:        "empty user",
			host:        "valid.example.com",
			port:        22,
			user:        "",
			keyPath:     "",
			timeout:     30 * time.Second,
			expectError: true,
			errorMsg:    "SSH user cannot be empty",
		},
		{
			name:        "negative timeout",
			host:        "valid.example.com",
			port:        22,
			user:        "validuser",
			keyPath:     "",
			timeout:     -5 * time.Second,
			expectError: true,
			errorMsg:    "SSH timeout cannot be negative",
		},
		{
			name:        "key path with directory traversal",
			host:        "valid.example.com",
			port:        22,
			user:        "validuser",
			keyPath:     "/path/../../../etc/passwd",
			timeout:     30 * time.Second,
			expectError: true,
			errorMsg:    "SSH key path cannot contain '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dokkuApi.NewSSHConfig(tt.host, tt.port, tt.user, tt.keyPath, tt.timeout)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
