package dokkuApi_test

import (
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
)

var _ = Describe("DokkuClient", func() {
	var (
		logger *slog.Logger
		config *dokkuApi.ClientConfig
		client dokkuApi.DokkuClient
	)

	BeforeEach(func() {
		logger = slog.Default()
		config = dokkuApi.DefaultClientConfig()
		client = dokkuApi.NewDokkuClient(config, logger)
	})

	Describe("Blacklist functionality", func() {
		Context("with no blacklist", func() {
			It("should allow commands", func() {
				client.SetBlacklist([]string{})

				err := client.ValidateCommand("apps:list", []string{})
				Expect(err).To(BeNil())
			})
		})

		Context("with exact match blacklist", func() {
			It("should block commands", func() {
				client.SetBlacklist([]string{"apps:destroy"})

				err := client.ValidateCommand("apps:destroy", []string{"myapp"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("blacklisted"))
			})
		})

		Context("with partial match blacklist", func() {
			It("should block commands", func() {
				client.SetBlacklist([]string{"destroy"})

				err := client.ValidateCommand("apps:destroy", []string{"myapp"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("blacklisted"))
			})
		})

		Context("with non-matching blacklist", func() {
			It("should allow commands", func() {
				client.SetBlacklist([]string{"delete", "remove"})

				err := client.ValidateCommand("apps:list", []string{})
				Expect(err).To(BeNil())
			})
		})

		Context("with multiple patterns", func() {
			It("should block if any match", func() {
				client.SetBlacklist([]string{"rm", "delete", "destroy"})

				err := client.ValidateCommand("apps:destroy", []string{"myapp"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("blacklisted"))
			})
		})
	})

	Describe("Security validation", func() {
		Context("with dangerous characters in command", func() {
			It("should block semicolon", func() {
				err := client.ValidateCommand("apps:list;rm -rf /", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})

			It("should block pipe", func() {
				err := client.ValidateCommand("apps:list|cat /etc/passwd", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})

			It("should block backtick", func() {
				err := client.ValidateCommand("apps:list`whoami`", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})

			It("should block dollar", func() {
				err := client.ValidateCommand("apps:list$(whoami)", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid character"))
			})
		})

		Context("with dangerous characters in args", func() {
			It("should block semicolon in args", func() {
				err := client.ValidateCommand("apps:list", []string{"myapp;rm -rf /"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported character"))
			})

			It("should block pipe in args", func() {
				err := client.ValidateCommand("apps:list", []string{"myapp|cat /etc/passwd"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported character"))
			})
		})

		Context("with characters the remote shell would reinterpret", func() {
			DescribeTable("rejects the argument",
				func(arg string) {
					Expect(client.ValidateCommand("config:show", []string{arg})).To(HaveOccurred())
				},
				Entry("space", "a b"),
				Entry("tab", "a\tb"),
				Entry("newline", "a\nb"),
				Entry("glob star", "*"),
				Entry("glob question mark", "file?"),
				Entry("bracket", "[ab]"),
				Entry("single quote", "it's"),
				Entry("double quote", `"x"`),
				Entry("backslash", `a\b`),
				Entry("leading hash", "#comment"),
				Entry("leading tilde", "~root"),
				Entry("empty", ""),
			)

			DescribeTable("accepts the argument",
				func(arg string) {
					Expect(client.ValidateCommand("config:show", []string{arg})).To(Succeed())
				},
				Entry("app name", "my-app"),
				Entry("scale pair", "web=2"),
				Entry("flag", "--no-restart"),
				Entry("base64 value", "KEY=aGVsbG8gd29ybGQ+Lz0="),
				Entry("git url", "https://github.com/dokku/smoke-test-app.git"),
				Entry("scp-style git url", "git@github.com:dokku/app.git"),
				Entry("buildpack with ref", "https://github.com/heroku/heroku-buildpack-go.git#v180"),
				Entry("image tag", "registry.example.com/team/app:1.2.3"),
				Entry("email", "ops@example.com"),
			)
		})
	})

	Describe("Allowlist functionality", func() {
		It("allows everything not blacklisted when empty", func() {
			client.SetAllowlist(nil)
			Expect(client.ValidateCommand("apps:create", []string{"x"})).To(Succeed())
		})

		It("only allows matching commands when set", func() {
			client.SetAllowlist([]string{"apps:list", "postgres:", "ps:*"})
			Expect(client.ValidateCommand("apps:list", nil)).To(Succeed())
			Expect(client.ValidateCommand("postgres:info", []string{"db"})).To(Succeed())
			Expect(client.ValidateCommand("ps:restart", []string{"app"})).To(Succeed())

			err := client.ValidateCommand("apps:destroy", []string{"app"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not in the allowlist"))
		})

		It("still applies the blacklist to allowlisted commands", func() {
			client.SetAllowlist([]string{"*"})
			client.SetBlacklist([]string{"destroy"})
			Expect(client.ValidateCommand("apps:destroy", []string{"app"})).To(HaveOccurred())
		})
	})
})
