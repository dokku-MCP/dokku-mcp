package shared_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
)

var _ = Describe("EnvVar", func() {
	Describe("NewEnvVarKey", func() {
		DescribeTable("creating a new EnvVarKey",
			func(key string, shouldFail bool, expectedError string) {
				k, err := shared.NewEnvVarKey(key)

				if shouldFail {
					Expect(err).To(HaveOccurred())
					Expect(k).To(BeNil())
					if expectedError != "" {
						Expect(err.Error()).To(Equal(expectedError))
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(k).ToNot(BeNil())
					if k != nil {
						Expect(k.Value()).To(Equal(key))
					}
				}
			},
			Entry("valid key", "MY_VAR", false, ""),
			Entry("valid key with numbers", "MY_VAR_1", false, ""),
			Entry("valid key starting with underscore", "_MY_VAR", false, ""),
			Entry("invalid key starting with a number", "1_MY_VAR", true, "invalid environment variable key format: 1_MY_VAR"),
			Entry("invalid key with hyphen", "MY-VAR", true, "invalid environment variable key format: MY-VAR"),
			Entry("invalid empty key", "", true, "invalid environment variable key format: "),
		)
	})

	Describe("Equal", func() {
		It("should correctly compare two EnvVarKey objects", func() {
			key1, _ := shared.NewEnvVarKey("MY_VAR")
			key2, _ := shared.NewEnvVarKey("MY_VAR")
			key3, _ := shared.NewEnvVarKey("OTHER_VAR")

			Expect(key1.Equal(key2)).To(BeTrue())
			Expect(key1.Equal(key3)).To(BeFalse())
			Expect(key1.Equal(nil)).To(BeFalse())
		})
	})

	Describe("NewEnvVarValue", func() {
		It("should create a new EnvVarValue", func() {
			value := shared.NewEnvVarValue("my value")
			Expect(value).ToNot(BeNil())
			Expect(value.Value()).To(Equal("my value"))
		})
	})

	Describe("Equal", func() {
		It("should correctly compare two EnvVarValue objects", func() {
			value1 := shared.NewEnvVarValue("my value")
			value2 := shared.NewEnvVarValue("my value")
			value3 := shared.NewEnvVarValue("other value")

			Expect(value1.Equal(value2)).To(BeTrue())
			Expect(value1.Equal(value3)).To(BeFalse())
			Expect(value1.Equal(nil)).To(BeFalse())
		})
	})
})
