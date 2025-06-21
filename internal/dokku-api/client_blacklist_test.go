package dokkuApi_test

import (
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
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
				Expect(err.Error()).To(ContainSubstring("dangerous character"))
			})

			It("should block pipe", func() {
				err := client.ValidateCommand("apps:list|cat /etc/passwd", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("dangerous character"))
			})

			It("should block backtick", func() {
				err := client.ValidateCommand("apps:list`whoami`", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("dangerous character"))
			})

			It("should block dollar", func() {
				err := client.ValidateCommand("apps:list$(whoami)", []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("dangerous character"))
			})
		})

		Context("with dangerous characters in args", func() {
			It("should block semicolon in args", func() {
				err := client.ValidateCommand("apps:list", []string{"myapp;rm -rf /"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("dangerous character"))
			})

			It("should block pipe in args", func() {
				err := client.ValidateCommand("apps:list", []string{"myapp|cat /etc/passwd"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("dangerous character"))
			})
		})
	})
})
