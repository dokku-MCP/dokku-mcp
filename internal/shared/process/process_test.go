package process_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alex-galey/dokku-mcp/internal/shared/process"
)

var _ = Describe("Process", func() {
	Describe("NewProcess", func() {
		It("should create a valid process", func() {
			proc, err := process.NewProcess(process.ProcessTypeWeb, "npm start", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(proc).ToNot(BeNil())
			Expect(proc.Type()).To(Equal(process.ProcessTypeWeb))
			Expect(proc.Command().Value()).To(Equal("npm start"))
			Expect(proc.Scale()).To(Equal(1))
			Expect(proc.HasCommand()).To(BeTrue())
		})

		It("should return an error for an invalid command", func() {
			proc, err := process.NewProcess(process.ProcessTypeWeb, "", 1)
			Expect(err).To(HaveOccurred())
			Expect(proc).To(BeNil())
			Expect(err.Error()).To(Equal("invalid process command: process command cannot be empty"))
		})

		It("should return an error for an invalid scale", func() {
			proc, err := process.NewProcess(process.ProcessTypeWeb, "npm start", -1)
			Expect(err).To(HaveOccurred())
			Expect(proc).To(BeNil())
			Expect(err.Error()).To(Equal("invalid process scale: process scale cannot be negative"))
		})
	})

	Describe("NewProcessForScaling", func() {
		It("should create a valid process without command", func() {
			proc, err := process.NewProcessForScaling(process.ProcessTypeWeb, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(proc).ToNot(BeNil())
			Expect(proc.Type()).To(Equal(process.ProcessTypeWeb))
			Expect(proc.Command()).To(BeNil())
			Expect(proc.Scale()).To(Equal(3))
			Expect(proc.HasCommand()).To(BeFalse())
		})

		It("should return an error for an invalid scale", func() {
			proc, err := process.NewProcessForScaling(process.ProcessTypeWeb, -1)
			Expect(err).To(HaveOccurred())
			Expect(proc).To(BeNil())
			Expect(err.Error()).To(Equal("invalid process scale: process scale cannot be negative"))
		})
	})

	Describe("SetScale", func() {
		var proc *process.Process

		BeforeEach(func() {
			proc, _ = process.NewProcess(process.ProcessTypeWeb, "npm start", 1)
		})

		It("should set a valid scale", func() {
			err := proc.SetScale(5)
			Expect(err).ToNot(HaveOccurred())
			Expect(proc.Scale()).To(Equal(5))
		})

		It("should return an error for an invalid scale and not change the value", func() {
			err := proc.SetScale(-1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("process scale cannot be negative"))
			Expect(proc.Scale()).To(Equal(1)) // Scale should not have changed
		})
	})

	Describe("SetCommand", func() {
		var proc *process.Process

		BeforeEach(func() {
			proc, _ = process.NewProcessForScaling(process.ProcessTypeWeb, 1)
		})

		It("should set a valid command", func() {
			err := proc.SetCommand("node server.js")
			Expect(err).ToNot(HaveOccurred())
			Expect(proc.Command().Value()).To(Equal("node server.js"))
			Expect(proc.HasCommand()).To(BeTrue())
		})

		It("should return an error for an empty command", func() {
			err := proc.SetCommand("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("process command cannot be empty"))
			Expect(proc.HasCommand()).To(BeFalse())
		})
	})
})
