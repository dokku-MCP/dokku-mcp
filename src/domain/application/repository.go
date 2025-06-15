package application

import (
	"context"
)

type Repository interface {
	GetAll(ctx context.Context) ([]*Application, error)
	GetByName(ctx context.Context, name string) (*Application, error)
	Save(ctx context.Context, app *Application) error
	Delete(ctx context.Context, name string) error
	Exists(ctx context.Context, name string) (bool, error)
	List(ctx context.Context, offset, limit int) ([]*Application, int, error)
	GetByState(ctx context.Context, state ApplicationState) ([]*Application, error)
}
