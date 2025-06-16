package service

import "context"

// Repository interface pour la persistance des services
type Repository interface {
	// Save sauvegarde un service
	Save(ctx context.Context, service *Service) error

	// GetByName récupère un service par son nom
	GetByName(ctx context.Context, name string) (*Service, error)

	// GetAll récupère tous les services
	GetAll(ctx context.Context) ([]*Service, error)

	// GetByType récupère tous les services d'un type donné
	GetByType(ctx context.Context, serviceType ServiceType) ([]*Service, error)

	// GetLinkedToApp récupère tous les services liés à une application
	GetLinkedToApp(ctx context.Context, appName string) ([]*Service, error)

	// Delete supprime un service
	Delete(ctx context.Context, name string) error

	// Exists vérifie si un service existe
	Exists(ctx context.Context, name string) (bool, error)
}
