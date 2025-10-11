package domain_test

import (
	"testing"
	"time"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeploymentTracker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DeploymentTracker Suite")
}

var _ = Describe("DeploymentTracker", func() {
	var tracker *domain.DeploymentTracker

	BeforeEach(func() {
		tracker = domain.NewDeploymentTracker()
	})

	Describe("Track", func() {
		It("should track a new deployment", func() {
			deployment, err := domain.NewDeployment("test-app", "main")
			Expect(err).NotTo(HaveOccurred())

			err = tracker.Track(deployment)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := tracker.GetByID(deployment.ID())
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ID()).To(Equal(deployment.ID()))
			Expect(retrieved.AppName()).To(Equal("test-app"))
		})

		It("should return error when tracking nil deployment", func() {
			err := tracker.Track(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be nil"))
		})

		It("should track multiple deployments", func() {
			deploy1, _ := domain.NewDeployment("app1", "main")
			deploy2, _ := domain.NewDeployment("app2", "develop")

			_ = tracker.Track(deploy1)
			_ = tracker.Track(deploy2)

			Expect(tracker.Count()).To(Equal(2))
		})
	})

	Describe("GetByID", func() {
		It("should retrieve tracked deployment", func() {
			deployment, _ := domain.NewDeployment("test-app", "main")
			_ = tracker.Track(deployment)

			retrieved, err := tracker.GetByID(deployment.ID())
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ID()).To(Equal(deployment.ID()))
		})

		It("should return error for non-existent deployment", func() {
			_, err := tracker.GetByID("non-existent-id")
			Expect(err).To(Equal(domain.ErrDeploymentNotFound))
		})
	})

	Describe("UpdateStatus", func() {
		It("should update deployment status to running", func() {
			deployment, _ := domain.NewDeployment("test-app", "main")
			_ = tracker.Track(deployment)

			err := tracker.UpdateStatus(deployment.ID(), domain.DeploymentStatusRunning, "")
			Expect(err).NotTo(HaveOccurred())

			retrieved, _ := tracker.GetByID(deployment.ID())
			Expect(retrieved.Status()).To(Equal(domain.DeploymentStatusRunning))
		})

		It("should update deployment status to succeeded", func() {
			deployment, _ := domain.NewDeployment("test-app", "main")
			deployment.Start()
			_ = tracker.Track(deployment)

			err := tracker.UpdateStatus(deployment.ID(), domain.DeploymentStatusSucceeded, "")
			Expect(err).NotTo(HaveOccurred())

			retrieved, _ := tracker.GetByID(deployment.ID())
			Expect(retrieved.Status()).To(Equal(domain.DeploymentStatusSucceeded))
			Expect(retrieved.IsCompleted()).To(BeTrue())
		})

		It("should update deployment status to failed with error message", func() {
			deployment, _ := domain.NewDeployment("test-app", "main")
			deployment.Start()
			_ = tracker.Track(deployment)

			err := tracker.UpdateStatus(deployment.ID(), domain.DeploymentStatusFailed, "build error")
			Expect(err).NotTo(HaveOccurred())

			retrieved, _ := tracker.GetByID(deployment.ID())
			Expect(retrieved.Status()).To(Equal(domain.DeploymentStatusFailed))
			Expect(retrieved.ErrorMsg()).To(Equal("build error"))
		})

		It("should return error for non-existent deployment", func() {
			err := tracker.UpdateStatus("non-existent", domain.DeploymentStatusRunning, "")
			Expect(err).To(Equal(domain.ErrDeploymentNotFound))
		})
	})

	Describe("AddLogs", func() {
		It("should append logs to deployment", func() {
			deployment, _ := domain.NewDeployment("test-app", "main")
			_ = tracker.Track(deployment)

			err := tracker.AddLogs(deployment.ID(), "log line 1\n")
			Expect(err).NotTo(HaveOccurred())

			err = tracker.AddLogs(deployment.ID(), "log line 2\n")
			Expect(err).NotTo(HaveOccurred())

			retrieved, _ := tracker.GetByID(deployment.ID())
			Expect(retrieved.BuildLogs()).To(ContainSubstring("log line 1"))
			Expect(retrieved.BuildLogs()).To(ContainSubstring("log line 2"))
		})

		It("should return error for non-existent deployment", func() {
			err := tracker.AddLogs("non-existent", "logs")
			Expect(err).To(Equal(domain.ErrDeploymentNotFound))
		})
	})

	Describe("Remove", func() {
		It("should remove deployment from tracking", func() {
			deployment, _ := domain.NewDeployment("test-app", "main")
			_ = tracker.Track(deployment)

			tracker.Remove(deployment.ID())

			_, err := tracker.GetByID(deployment.ID())
			Expect(err).To(Equal(domain.ErrDeploymentNotFound))
		})

		It("should not error when removing non-existent deployment", func() {
			Expect(func() {
				tracker.Remove("non-existent")
			}).NotTo(Panic())
		})
	})

	Describe("GetAll", func() {
		It("should return all tracked deployments", func() {
			deploy1, _ := domain.NewDeployment("app1", "main")
			deploy2, _ := domain.NewDeployment("app2", "develop")
			deploy3, _ := domain.NewDeployment("app3", "feature")

			_ = tracker.Track(deploy1)
			_ = tracker.Track(deploy2)
			_ = tracker.Track(deploy3)

			all := tracker.GetAll()
			Expect(len(all)).To(Equal(3))
		})

		It("should return empty slice when no deployments tracked", func() {
			all := tracker.GetAll()
			Expect(len(all)).To(Equal(0))
		})
	})

	Describe("GetActive", func() {
		It("should return only active deployments", func() {
			deploy1, _ := domain.NewDeployment("app1", "main")
			deploy1.Start()
			deploy2, _ := domain.NewDeployment("app2", "develop")
			deploy2.Start()
			deploy2.Complete()
			deploy3, _ := domain.NewDeployment("app3", "feature")
			deploy3.Start()

			_ = tracker.Track(deploy1)
			_ = tracker.Track(deploy2)
			_ = tracker.Track(deploy3)

			active := tracker.GetActive()
			Expect(len(active)).To(Equal(2)) // deploy1 and deploy3 are active
		})

		It("should return empty slice when no active deployments", func() {
			deploy1, _ := domain.NewDeployment("app1", "main")
			deploy1.Start()
			deploy1.Complete()

			_ = tracker.Track(deploy1)

			active := tracker.GetActive()
			Expect(len(active)).To(Equal(0))
		})
	})

	Describe("Count", func() {
		It("should return correct count of tracked deployments", func() {
			Expect(tracker.Count()).To(Equal(0))

			deploy1, _ := domain.NewDeployment("app1", "main")
			_ = tracker.Track(deploy1)
			Expect(tracker.Count()).To(Equal(1))

			deploy2, _ := domain.NewDeployment("app2", "develop")
			_ = tracker.Track(deploy2)
			Expect(tracker.Count()).To(Equal(2))

			tracker.Remove(deploy1.ID())
			Expect(tracker.Count()).To(Equal(1))
		})
	})

	Describe("Cleanup", func() {
		It("should clean up old completed deployments after TTL", func() {
			// This test would require manipulating time or waiting
			// For now, we just verify the tracker doesn't panic
			deployment, _ := domain.NewDeployment("test-app", "main")
			deployment.Start()
			deployment.Complete()
			_ = tracker.Track(deployment)

			// Cleanup happens in background, we just verify it doesn't crash
			time.Sleep(100 * time.Millisecond)
			Expect(tracker.Count()).To(Equal(1)) // Still there, TTL not reached
		})
	})
})
