package app

import (
	"context"
)

type ApplicationRepository interface {
	Save(ctx context.Context, app *Application) error
	GetByName(ctx context.Context, name *ApplicationName) (*Application, error)
	GetAll(ctx context.Context) ([]*Application, error)
	GetByState(ctx context.Context, state *ApplicationState) ([]*Application, error)
	Delete(ctx context.Context, name *ApplicationName) error
	Exists(ctx context.Context, name *ApplicationName) (bool, error)
	List(ctx context.Context, offset, limit int) ([]*Application, int, error)
	GetByDomain(ctx context.Context, domain string) ([]*Application, error)
	GetRunningApplications(ctx context.Context) ([]*Application, error)
	GetApplicationsWithBuildpack(ctx context.Context, buildpack string) ([]*Application, error)
	GetRecentlyDeployed(ctx context.Context, limit int) ([]*Application, error)
	CountByState(ctx context.Context) (map[StateValue]int, error)
	GetApplicationMetrics(ctx context.Context) (*ApplicationMetrics, error)
}

type ApplicationMetrics struct {
	TotalApplications     int
	RunningApplications   int
	StoppedApplications   int
	ErrorApplications     int
	DeployingApplications int
	TotalDeployments      int
	SuccessfulDeployments int
	FailedDeployments     int
	AverageDeploymentTime float64 // en secondes
	MostUsedBuildpacks    map[string]int
	ApplicationsByState   map[StateValue]int
}

type ApplicationQuery struct {
	State         *ApplicationState
	Buildpack     string
	Domain        string
	HasDomain     bool
	CreatedAfter  *string
	CreatedBefore *string

	SortBy    ApplicationSortField
	SortOrder SortOrder

	Offset int
	Limit  int

	IncludeConfiguration  bool
	IncludeDeploymentInfo bool
	IncludeEvents         bool
}

type ApplicationSortField string

const (
	SortByName       ApplicationSortField = "name"
	SortByCreatedAt  ApplicationSortField = "created_at"
	SortByUpdatedAt  ApplicationSortField = "updated_at"
	SortByState      ApplicationSortField = "state"
	SortByLastDeploy ApplicationSortField = "last_deploy"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type QueryableApplicationRepository interface {
	ApplicationRepository
	Query(ctx context.Context, query *ApplicationQuery) ([]*Application, int, error)
	Search(ctx context.Context, searchTerm string, limit int) ([]*Application, error)
	GetApplicationsRequiringAttention(ctx context.Context) ([]*Application, error)
}
