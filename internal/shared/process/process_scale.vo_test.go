package process_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dokku-mcp/dokku-mcp/internal/shared/process"
)

var _ = Describe("ProcessScale", func() {
	Describe("NewProcessScale", func() {
		DescribeTable("creating a new ProcessScale",
			func(scale int, shouldFail bool, expectedError string) {
				s, err := process.NewProcessScale(scale)

				if shouldFail {
					Expect(err).To(HaveOccurred())
					Expect(s).To(BeNil())
					if expectedError != "" {
						Expect(err.Error()).To(Equal(expectedError))
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(s).ToNot(BeNil())
					if s != nil {
						Expect(s.Value()).To(Equal(scale))
					}
				}
			},
			Entry("valid scale", 1, false, ""),
			Entry("zero scale", 0, false, ""),
			Entry("invalid negative scale", -1, true, "process scale cannot be negative"),
		)
	})

	Describe("Equal", func() {
		It("should correctly compare two ProcessScale objects", func() {
			scale1, _ := process.NewProcessScale(1)
			scale2, _ := process.NewProcessScale(1)
			scale3, _ := process.NewProcessScale(2)

			Expect(scale1.Equal(scale2)).To(BeTrue())
			Expect(scale1.Equal(scale3)).To(BeFalse())
			Expect(scale1.Equal(nil)).To(BeFalse())
		})
	})
})
