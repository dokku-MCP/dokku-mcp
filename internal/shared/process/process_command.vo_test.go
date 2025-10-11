package process_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dokku-mcp/dokku-mcp/internal/shared/process"
)

var _ = Describe("ProcessCommand", func() {
	Describe("NewProcessCommand", func() {
		DescribeTable("creating a new ProcessCommand",
			func(command string, shouldFail bool, expectedError string) {
				cmd, err := process.NewProcessCommand(command)

				if shouldFail {
					Expect(err).To(HaveOccurred())
					Expect(cmd).To(BeNil())
					if expectedError != "" {
						Expect(err.Error()).To(Equal(expectedError))
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(cmd).ToNot(BeNil())
					if cmd != nil {
						Expect(cmd.Value()).To(Equal(strings.TrimSpace(command)))
					}
				}
			},
			Entry("valid command", "npm start", false, ""),
			Entry("valid command with spaces to trim", "  npm start  ", false, ""),
			Entry("invalid empty command", "", true, "process command cannot be empty"),
			Entry("invalid blank command", "   ", true, "process command cannot be empty"),
		)
	})

	Describe("Equal", func() {
		It("should correctly compare two ProcessCommand objects", func() {
			cmd1, _ := process.NewProcessCommand("npm start")
			cmd2, _ := process.NewProcessCommand("npm start")
			cmd3, _ := process.NewProcessCommand("npm run dev")

			Expect(cmd1.Equal(cmd2)).To(BeTrue())
			Expect(cmd1.Equal(cmd3)).To(BeFalse())
			Expect(cmd1.Equal(nil)).To(BeFalse())
		})
	})
})
