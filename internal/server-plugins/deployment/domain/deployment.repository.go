package domain

import "context"

type DeploymentRepository interface {
	Save(ctx context.Context, deployment *Deployment) error
	FindByID(ctx context.Context, id string) (*Deployment, error)
	FindByAppName(ctx context.Context, appName string) ([]*Deployment, error)
	FindAll(ctx context.Context) ([]*Deployment, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, deployment *Deployment) error
}
