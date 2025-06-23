package process_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alex-galey/dokku-mcp/internal/shared/process"
)

var _ = Describe("RestartPolicy", func() {
	Describe("NewRestartPolicyFromString", func() {
		DescribeTable("parsing a restart policy string",
			func(policyStr string, shouldFail bool, expectedError string, expectedType process.RestartPolicyType, expectedMaxRestarts int) {
				policy, err := process.NewRestartPolicyFromString(policyStr)

				if shouldFail {
					Expect(err).To(HaveOccurred())
					Expect(policy).To(BeNil())
					if expectedError != "" {
						Expect(err.Error()).To(Equal(expectedError))
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(policy).ToNot(BeNil())
					if policy != nil {
						Expect(policy.Policy()).To(Equal(expectedType))
						Expect(policy.MaxRestarts()).To(Equal(expectedMaxRestarts))
					}
				}
			},
			Entry("valid on-failure with default restarts", "on-failure", false, "", process.RestartPolicyOnFailure, 10),
			Entry("valid on-failure with specific restarts", "on-failure:5", false, "", process.RestartPolicyOnFailure, 5),
			Entry("valid always", "always", false, "", process.RestartPolicyAlways, 10),
			Entry("valid never", "never", false, "", process.RestartPolicyNever, 10),
			Entry("valid 'no' as never", "no", false, "", process.RestartPolicyNever, 10),
			Entry("valid unless-stopped", "unless-stopped", false, "", process.RestartPolicyUnlessStopped, 10),
			Entry("invalid empty string", "", true, "restart policy string cannot be empty", process.RestartPolicyType(""), 0),
			Entry("invalid policy type", "sometimes", true, "invalid restart policy type: sometimes", process.RestartPolicyType(""), 0),
			Entry("invalid on-failure format", "on-failure:abc", true, "invalid max restarts count in policy: on-failure:abc", process.RestartPolicyType(""), 0),
			Entry("invalid policy with max restarts", "always:5", true, "max restarts count is only applicable for 'on-failure' policy: always:5", process.RestartPolicyType(""), 0),
		)
	})
})
