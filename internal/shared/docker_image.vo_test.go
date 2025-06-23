package shared_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alex-galey/dokku-mcp/internal/shared"
)

var _ = Describe("DockerImage", func() {
	Describe("NewDockerImage", func() {
		DescribeTable("creating a new Docker image",
			func(image string, shouldFail bool, expectedError string) {
				di, err := shared.NewDockerImage(image)

				if shouldFail {
					Expect(err).To(HaveOccurred())
					Expect(di).To(BeNil())
					if expectedError != "" {
						Expect(err.Error()).To(Equal(expectedError))
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(di).ToNot(BeNil())
					if di != nil {
						Expect(di.Value()).To(Equal(image))
					}
				}
			},
			Entry("valid image name", "ubuntu:latest", false, ""),
			Entry("valid image without tag", "ubuntu", false, ""),
			Entry("valid image with user", "user/ubuntu:latest", false, ""),
			Entry("valid image with registry", "gcr.io/user/ubuntu:latest", false, ""),
			Entry("valid image with numbers", "test-123/app:v1.2.3", false, ""),
			Entry("valid image with dots", "my.registry/user/app:latest", false, ""),
			Entry("invalid image name with spaces", "ubuntu image", true, "invalid docker image format: ubuntu image"),
			Entry("invalid image name with uppercase", "Ubuntu:latest", true, "invalid docker image format: Ubuntu:latest"),
			Entry("invalid image name with trailing slash", "user/app/:latest", true, "invalid docker image format: user/app/:latest"),
		)
	})

	Describe("Equal", func() {
		It("should correctly compare two DockerImage objects", func() {
			image1, _ := shared.NewDockerImage("ubuntu:latest")
			image2, _ := shared.NewDockerImage("ubuntu:latest")
			image3, _ := shared.NewDockerImage("alpine:latest")

			Expect(image1.Equal(image2)).To(BeTrue())
			Expect(image1.Equal(image3)).To(BeFalse())
			Expect(image1.Equal(nil)).To(BeFalse())
		})
	})
})
