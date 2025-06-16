package dokku

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DokkuClient - SSH Authentication", func() {
	var (
		authService *SSHAuthService
		logger      *slog.Logger
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(GinkgoWriter, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	})

	// Force cleanup of any hanging processes after each test
	AfterEach(func() {
		if authService != nil {
			// Allow time for any cleanup operations
			Eventually(func() bool {
				return true
			}, "1s", "100ms").Should(BeTrue())
		}
	})

	Describe("DetermineAuthMethod", func() {
		Context("when ssh-agent is available", func() {
			BeforeEach(func() {
				config := &SSHAuthConfig{
					CheckAgent: func() bool { return true },
				}
				authService = NewSSHAuthServiceWithConfig(logger, config)
			})

			It("should use ssh-agent when available", func() {
				authMethod := authService.DetermineAuthMethod("")
				Expect(authMethod).NotTo(BeNil())
				Expect(authMethod.Description).To(ContainSubstring("ssh-agent"))
				Expect(authMethod.UseAgent).To(BeTrue())
			})
		})

		Context("when ssh-agent is not available", func() {
			Context("and ~/.ssh/id_rsa exists", func() {
				var tempHomeDir string
				var tempKeyPath string

				BeforeEach(func() {
					var err error
					tempHomeDir, err = os.MkdirTemp("", "test-home-*")
					Expect(err).NotTo(HaveOccurred())

					// Create .ssh directory
					sshDir := filepath.Join(tempHomeDir, ".ssh")
					err = os.MkdirAll(sshDir, 0700)
					Expect(err).NotTo(HaveOccurred())

					// Create fake key file
					tempKeyPath = filepath.Join(sshDir, "id_rsa")
					err = os.WriteFile(tempKeyPath, []byte("fake-key-content"), 0600)
					Expect(err).NotTo(HaveOccurred())

					config := &SSHAuthConfig{
						HomeDir:    tempHomeDir,
						CheckAgent: func() bool { return false }, // ssh-agent not available
					}
					authService = NewSSHAuthServiceWithConfig(logger, config)

					DeferCleanup(func() {
						// Ensure proper cleanup with timeout
						done := make(chan bool, 1)
						go func() {
							os.RemoveAll(tempHomeDir)
							done <- true
						}()

						select {
						case <-done:
							// Cleanup completed
						case <-time.After(2 * time.Second):
							// Timeout, but continue to avoid hanging
							GinkgoWriter.Printf("Timeout during cleanup of %s\n", tempHomeDir)
						}
					})
				})

				It("should use ~/.ssh/id_rsa as fallback", func() {
					authMethod := authService.DetermineAuthMethod("")
					Expect(authMethod).NotTo(BeNil())
					Expect(authMethod.UseAgent).To(BeFalse())
					Expect(authMethod.KeyPath).To(Equal(tempKeyPath))
					Expect(authMethod.Description).To(ContainSubstring("default key"))
				})
			})

			Context("and ~/.ssh/id_rsa does not exist", func() {
				Context("but a key is configured", func() {
					var tempKeyPath string

					BeforeEach(func() {
						// Create temporary key file
						tempFile, err := os.CreateTemp("", "test-key-*")
						Expect(err).NotTo(HaveOccurred())
						tempKeyPath = tempFile.Name()
						_, err = tempFile.WriteString("fake-key-content")
						Expect(err).NotTo(HaveOccurred())
						tempFile.Close()

						tempHomeDir, err := os.MkdirTemp("", "test-empty-home-*")
						Expect(err).NotTo(HaveOccurred())

						config := &SSHAuthConfig{
							HomeDir:    tempHomeDir,
							CheckAgent: func() bool { return false },
						}
						authService = NewSSHAuthServiceWithConfig(logger, config)

						DeferCleanup(func() {
							os.Remove(tempKeyPath)
							os.RemoveAll(tempHomeDir)
						})
					})

					It("should use the configured key", func() {
						authMethod := authService.DetermineAuthMethod(tempKeyPath)
						Expect(authMethod).NotTo(BeNil())
						Expect(authMethod.UseAgent).To(BeFalse())
						Expect(authMethod.KeyPath).To(Equal(tempKeyPath))
						Expect(authMethod.Description).To(ContainSubstring("configured key"))
					})
				})

				Context("and no key is configured", func() {
					BeforeEach(func() {
						tempHomeDir, err := os.MkdirTemp("", "test-empty-home-*")
						Expect(err).NotTo(HaveOccurred())

						config := &SSHAuthConfig{
							HomeDir:    tempHomeDir,
							CheckAgent: func() bool { return false },
						}
						authService = NewSSHAuthServiceWithConfig(logger, config)

						DeferCleanup(func() {
							os.RemoveAll(tempHomeDir)
						})
					})

					It("should use ssh-agent as fallback", func() {
						authMethod := authService.DetermineAuthMethod("")
						Expect(authMethod).NotTo(BeNil())
						Expect(authMethod.UseAgent).To(BeTrue())
						Expect(authMethod.Description).To(ContainSubstring("fallback"))
					})
				})
			})
		})
	})

	Describe("isKeyFileAccessible", func() {
		BeforeEach(func() {
			authService = NewSSHAuthService(logger)
		})

		Context("with a valid key file", func() {
			var tempKeyPath string

			BeforeEach(func() {
				tempFile, err := os.CreateTemp("", "test-key-*")
				Expect(err).NotTo(HaveOccurred())
				tempKeyPath = tempFile.Name()
				_, err = tempFile.WriteString("fake-key-content")
				Expect(err).NotTo(HaveOccurred())
				tempFile.Close()

				DeferCleanup(func() {
					os.Remove(tempKeyPath)
				})
			})

			It("should return true", func() {
				result := authService.isKeyFileAccessible(tempKeyPath)
				Expect(result).To(BeTrue())
			})
		})

		Context("with an inexistent file", func() {
			It("should return false", func() {
				result := authService.isKeyFileAccessible("/nonexistent/key/path")
				Expect(result).To(BeFalse())
			})
		})

		Context("with an empty path", func() {
			It("should return false", func() {
				result := authService.isKeyFileAccessible("")
				Expect(result).To(BeFalse())
			})
		})

		Context("with a path using tilde", func() {
			var tempHomeDir string
			var tempKeyPath string

			BeforeEach(func() {
				var err error
				tempHomeDir, err = os.MkdirTemp("", "test-home-*")
				Expect(err).NotTo(HaveOccurred())

				// Create a key file in the temporary directory
				tempKeyPath = filepath.Join(tempHomeDir, "test-key")
				err = os.WriteFile(tempKeyPath, []byte("fake-key-content"), 0600)
				Expect(err).NotTo(HaveOccurred())

				config := &SSHAuthConfig{
					HomeDir: tempHomeDir,
				}
				authService = NewSSHAuthServiceWithConfig(logger, config)

				DeferCleanup(func() {
					os.RemoveAll(tempHomeDir)
				})
			})

			It("should expand the tilde correctly", func() {
				result := authService.isKeyFileAccessible("~/test-key")
				Expect(result).To(BeTrue())
			})
		})
	})

	Describe("isSshAgentAvailable", func() {
		BeforeEach(func() {
			authService = NewSSHAuthService(logger)
		})

		Context("when SSH_AUTH_SOCK is not defined", func() {
			BeforeEach(func() {
				os.Unsetenv("SSH_AUTH_SOCK")
			})

			It("should return false", func() {
				result := authService.isSshAgentAvailable()
				Expect(result).To(BeFalse())
			})
		})

		Context("when SSH_AUTH_SOCK points to an inexistent socket", func() {
			BeforeEach(func() {
				os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket")
			})

			AfterEach(func() {
				os.Unsetenv("SSH_AUTH_SOCK")
			})

			It("should return false", func() {
				result := authService.isSshAgentAvailable()
				Expect(result).To(BeFalse())
			})
		})
	})
})

// Functional tests for SSH authentication logic
func TestSSHAuthenticationLogic(t *testing.T) {
	tests := []struct {
		name                string
		sshAgentAvailable   bool
		homeSSHKeyExists    bool
		configSSHKeyPath    string
		configSSHKeyExists  bool
		expectedUseAgent    bool
		expectedDescription string
	}{
		{
			name:                "ssh-agent available",
			sshAgentAvailable:   true,
			expectedUseAgent:    true,
			expectedDescription: "ssh-agent",
		},
		{
			name:                "fallback to ~/.ssh/id_rsa",
			sshAgentAvailable:   false,
			homeSSHKeyExists:    true,
			expectedUseAgent:    false,
			expectedDescription: "default key",
		},
		{
			name:                "use the configured key",
			sshAgentAvailable:   false,
			homeSSHKeyExists:    false,
			configSSHKeyPath:    "/tmp/configured-key",
			configSSHKeyExists:  true,
			expectedUseAgent:    false,
			expectedDescription: "configured key",
		},
		{
			name:                "last resort - ssh-agent",
			sshAgentAvailable:   false,
			homeSSHKeyExists:    false,
			configSSHKeyPath:    "/nonexistent/key",
			configSSHKeyExists:  false,
			expectedUseAgent:    true,
			expectedDescription: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary home directory
			tempHomeDir, err := os.MkdirTemp("", "test-home-*")
			if err != nil {
				t.Fatalf("Failed to create temp home dir: %v", err)
			}
			defer os.RemoveAll(tempHomeDir)

			// Create ~/.ssh/id_rsa if necessary
			if tt.homeSSHKeyExists {
				sshDir := filepath.Join(tempHomeDir, ".ssh")
				if err := os.MkdirAll(sshDir, 0700); err != nil {
					t.Fatalf("Failed to create .ssh dir: %v", err)
				}
				keyPath := filepath.Join(sshDir, "id_rsa")
				if err := os.WriteFile(keyPath, []byte("fake-key"), 0600); err != nil {
					t.Fatalf("Failed to create home SSH key: %v", err)
				}
			}

			// Create the configured key if necessary
			var configKeyPath string
			if tt.configSSHKeyExists && tt.configSSHKeyPath != "" {
				tempFile, err := os.CreateTemp("", "config-key-*")
				if err != nil {
					t.Fatalf("Failed to create config key: %v", err)
				}
				configKeyPath = tempFile.Name()
				_, err = tempFile.WriteString("fake-config-key")
				if err != nil {
					t.Fatalf("Failed to write config key: %v", err)
				}
				tempFile.Close()
				defer os.Remove(configKeyPath)
			} else {
				configKeyPath = tt.configSSHKeyPath
			}

			// Create the authentication service
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			config := &SSHAuthConfig{
				HomeDir: tempHomeDir,
				CheckAgent: func() bool {
					return tt.sshAgentAvailable
				},
			}

			authService := NewSSHAuthServiceWithConfig(logger, config)

			// Test the method
			authMethod := authService.DetermineAuthMethod(configKeyPath)

			// Verify results
			if authMethod.UseAgent != tt.expectedUseAgent {
				t.Errorf("Expected UseAgent=%v, got %v", tt.expectedUseAgent, authMethod.UseAgent)
			}

			if !containsSubstring(authMethod.Description, tt.expectedDescription) {
				t.Errorf("Expected description to contain '%s', got '%s'", tt.expectedDescription, authMethod.Description)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
