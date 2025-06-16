package service

import (
	"fmt"
	"time"
)

// ServiceType représente le type de service
type ServiceType string

const (
	ServiceTypePostgreSQL    ServiceType = "postgres"
	ServiceTypeMySQL         ServiceType = "mysql"
	ServiceTypeRedis         ServiceType = "redis"
	ServiceTypeMongoDB       ServiceType = "mongo"
	ServiceTypeRabbitMQ      ServiceType = "rabbitmq"
	ServiceTypeElasticSearch ServiceType = "elasticsearch"
	ServiceTypeMinio         ServiceType = "minio"
	ServiceTypeMemcached     ServiceType = "memcached"
)

// ServiceState représente l'état du service
type ServiceState string

const (
	ServiceStateCreated  ServiceState = "created"
	ServiceStateStarting ServiceState = "starting"
	ServiceStateRunning  ServiceState = "running"
	ServiceStateStopped  ServiceState = "stopped"
	ServiceStateError    ServiceState = "error"
)

// ServiceVersion représente une version de service
type ServiceVersion struct {
	Major int
	Minor int
	Patch int
}

func (v ServiceVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ServiceConfiguration contient la configuration d'un service
type ServiceConfiguration struct {
	Image       string            `json:"image"`
	Environment map[string]string `json:"environment"`
	Ports       map[string]int    `json:"ports"`
	Volumes     []VolumeMount     `json:"volumes"`
	Memory      string            `json:"memory,omitempty"`
	CPU         string            `json:"cpu,omitempty"`
}

// VolumeMount représente un montage de volume
type VolumeMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
}

// Service représente un service Dokku (base de données, cache, etc.)
type Service struct {
	name          string
	serviceType   ServiceType
	version       ServiceVersion
	state         ServiceState
	config        *ServiceConfiguration
	createdAt     time.Time
	updatedAt     time.Time
	linkedApps    []string
	connectionURL string
}

// NewService crée un nouveau service
func NewService(name string, serviceType ServiceType, version ServiceVersion) (*Service, error) {
	if name == "" {
		return nil, fmt.Errorf("le nom du service ne peut pas être vide")
	}

	if len(name) > 63 {
		return nil, fmt.Errorf("le nom du service ne peut pas dépasser 63 caractères")
	}

	now := time.Now()
	return &Service{
		name:        name,
		serviceType: serviceType,
		version:     version,
		state:       ServiceStateCreated,
		config:      NewServiceConfiguration(),
		createdAt:   now,
		updatedAt:   now,
		linkedApps:  make([]string, 0),
	}, nil
}

// NewServiceConfiguration crée une nouvelle configuration de service
func NewServiceConfiguration() *ServiceConfiguration {
	return &ServiceConfiguration{
		Environment: make(map[string]string),
		Ports:       make(map[string]int),
		Volumes:     make([]VolumeMount, 0),
	}
}

// Getters
func (s *Service) Name() string                  { return s.name }
func (s *Service) ServiceType() ServiceType      { return s.serviceType }
func (s *Service) Version() ServiceVersion       { return s.version }
func (s *Service) State() ServiceState           { return s.state }
func (s *Service) Config() *ServiceConfiguration { return s.config }
func (s *Service) CreatedAt() time.Time          { return s.createdAt }
func (s *Service) UpdatedAt() time.Time          { return s.updatedAt }
func (s *Service) LinkedApps() []string          { return s.linkedApps }
func (s *Service) ConnectionURL() string         { return s.connectionURL }

// UpdateState met à jour l'état du service
func (s *Service) UpdateState(newState ServiceState) error {
	if !s.isValidStateTransition(s.state, newState) {
		return fmt.Errorf("transition d'état invalide de %s vers %s", s.state, newState)
	}

	s.state = newState
	s.updatedAt = time.Now()
	return nil
}

// Start démarre le service
func (s *Service) Start() error {
	return s.UpdateState(ServiceStateStarting)
}

// Stop arrête le service
func (s *Service) Stop() error {
	return s.UpdateState(ServiceStateStopped)
}

// LinkToApp lie le service à une application
func (s *Service) LinkToApp(appName string) error {
	if appName == "" {
		return fmt.Errorf("le nom de l'application ne peut pas être vide")
	}

	// Vérifier si déjà lié
	for _, linkedApp := range s.linkedApps {
		if linkedApp == appName {
			return fmt.Errorf("le service est déjà lié à l'application %s", appName)
		}
	}

	s.linkedApps = append(s.linkedApps, appName)
	s.updatedAt = time.Now()
	return nil
}

// UnlinkFromApp délie le service d'une application
func (s *Service) UnlinkFromApp(appName string) error {
	for i, linkedApp := range s.linkedApps {
		if linkedApp == appName {
			s.linkedApps = append(s.linkedApps[:i], s.linkedApps[i+1:]...)
			s.updatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("le service n'est pas lié à l'application %s", appName)
}

// IsLinkedToApp vérifie si le service est lié à une application
func (s *Service) IsLinkedToApp(appName string) bool {
	for _, linkedApp := range s.linkedApps {
		if linkedApp == appName {
			return true
		}
	}
	return false
}

// SetConnectionURL définit l'URL de connexion
func (s *Service) SetConnectionURL(url string) {
	s.connectionURL = url
	s.updatedAt = time.Now()
}

// UpdateConfiguration met à jour la configuration
func (s *Service) UpdateConfiguration(config *ServiceConfiguration) error {
	if config == nil {
		return fmt.Errorf("la configuration ne peut pas être nulle")
	}

	s.config = config
	s.updatedAt = time.Now()
	return nil
}

// IsRunning vérifie si le service est en cours d'exécution
func (s *Service) IsRunning() bool {
	return s.state == ServiceStateRunning
}

// GetDefaultPort retourne le port par défaut selon le type de service
func (s *Service) GetDefaultPort() int {
	switch s.serviceType {
	case ServiceTypePostgreSQL:
		return 5432
	case ServiceTypeMySQL:
		return 3306
	case ServiceTypeRedis:
		return 6379
	case ServiceTypeMongoDB:
		return 27017
	case ServiceTypeRabbitMQ:
		return 5672
	case ServiceTypeElasticSearch:
		return 9200
	case ServiceTypeMinio:
		return 9000
	case ServiceTypeMemcached:
		return 11211
	default:
		return 0
	}
}

// isValidStateTransition vérifie si la transition d'état est valide
func (s *Service) isValidStateTransition(from, to ServiceState) bool {
	validTransitions := map[ServiceState][]ServiceState{
		ServiceStateCreated:  {ServiceStateStarting, ServiceStateError},
		ServiceStateStarting: {ServiceStateRunning, ServiceStateError},
		ServiceStateRunning:  {ServiceStateStopped, ServiceStateError},
		ServiceStateStopped:  {ServiceStateStarting, ServiceStateError},
		ServiceStateError:    {ServiceStateCreated, ServiceStateStarting, ServiceStateStopped},
	}

	validToStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, validTo := range validToStates {
		if validTo == to {
			return true
		}
	}

	return false
}
