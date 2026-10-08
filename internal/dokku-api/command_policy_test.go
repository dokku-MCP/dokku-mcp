package dokkuApi_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
)

var _ = Describe("Command policy", func() {
	DescribeTable("MatchesCommandPattern",
		func(command, pattern string, expected bool) {
			Expect(dokkuApi.MatchesCommandPattern(command, pattern)).To(Equal(expected))
		},
		Entry("exact", "apps:list", "apps:list", true),
		Entry("exact mismatch", "apps:list", "apps:lis", false),
		Entry("colon prefix", "postgres:create", "postgres:", true),
		Entry("colon prefix mismatch", "postgresql:create", "postgres:", false),
		Entry("star prefix", "apps:report", "apps:*", true),
		Entry("star everything", "anything", "*", true),
	)

	DescribeTable("IsCacheableCommand",
		func(command string, expected bool) {
			Expect(dokkuApi.IsCacheableCommand(command)).To(Equal(expected))
		},
		Entry("report", "apps:report", true),
		Entry("list", "apps:list", true),
		Entry("show", "config:show", true),
		Entry("version", "version", true),
		Entry("service links", "postgres:links", true),
		Entry("logs are always fresh", "logs", false),
		Entry("create", "apps:create", false),
		Entry("config set", "config:set", false),
		Entry("deploy", "git:sync", false),
		Entry("rebuild", "ps:rebuild", false),
	)
})
