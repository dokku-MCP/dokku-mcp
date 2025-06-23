package app_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	app "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/domain"
)

var _ = Describe("ApplicationCommand", func() {
	Describe("IsValid", func() {
		Context("with valid commands", func() {
			It("should return true for all allowed commands", func() {
				validCommands := []app.ApplicationCommand{
					app.CommandAppsList,
					app.CommandAppsInfo,
					app.CommandAppsCreate,
					app.CommandAppsDestroy,
					app.CommandAppsExists,
					app.CommandAppsReport,
					app.CommandConfigShow,
					app.CommandConfigSet,
					app.CommandPsScale,
					app.CommandPsReport,
					app.CommandLogs,
				}

				for _, cmd := range validCommands {
					Expect(cmd.IsValid()).To(BeTrue(), "Command %s should be valid", cmd)
				}
			})
		})

		Context("with invalid commands", func() {
			It("should return false for non-whitelisted commands", func() {
				invalidCommands := []app.ApplicationCommand{
					"rm -rf /",
					"apps:delete",
					"sudo reboot",
					"git:push",
					"domains:add",
				}

				for _, cmd := range invalidCommands {
					Expect(cmd.IsValid()).To(BeFalse(), "Command %s should be invalid", cmd)
				}
			})
		})
	})

	Describe("String", func() {
		It("should return the string representation of the command", func() {
			cmd := app.CommandAppsList
			Expect(cmd.String()).To(Equal("apps:list"))
		})
	})

	Describe("GetAllowedCommands", func() {
		It("should return all allowed commands", func() {
			commands := app.GetAllowedCommands()
			Expect(commands).To(HaveLen(11))
			Expect(commands).To(ContainElements(
				app.CommandAppsList,
				app.CommandAppsInfo,
				app.CommandAppsCreate,
				app.CommandAppsDestroy,
				app.CommandAppsExists,
				app.CommandAppsReport,
				app.CommandConfigShow,
				app.CommandConfigSet,
				app.CommandPsScale,
				app.CommandPsReport,
				app.CommandLogs,
			))
		})
	})
})
