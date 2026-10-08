package app

import (
	"context"
	"strings"
)

type ApplicationRepository interface {
	Save(ctx context.Context, app *Application) error
	GetByName(ctx context.Context, name *ApplicationName) (*Application, error)
	GetAll(ctx context.Context) ([]*Application, error)
	GetByState(ctx context.Context, state *ApplicationState) ([]*Application, error)
	Delete(ctx context.Context, name *ApplicationName) error
	Exists(ctx context.Context, name *ApplicationName) (bool, error)
	// GetLogs returns the last lines of the application's runtime logs.
	GetLogs(ctx context.Context, name *ApplicationName, lines int) (string, error)
	// FollowLogs streams new runtime log lines until ctx is cancelled. The
	// line channel is closed when the stream ends; the error channel then
	// yields at most one error that ended the stream early.
	FollowLogs(ctx context.Context, name *ApplicationName) (<-chan string, <-chan error, error)
	// GetFailedDeployLogs returns the logs of containers from the last failed deploy.
	GetFailedDeployLogs(ctx context.Context, name *ApplicationName) (string, error)
	// SetProcessState restarts, stops or starts all processes of the application.
	SetProcessState(ctx context.Context, name *ApplicationName, action ProcessAction) error
	// GetDeploySource returns how the application was last deployed.
	GetDeploySource(ctx context.Context, name *ApplicationName) (DeploySource, error)
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

// ProcessAction is a lifecycle operation on all processes of an application.
type ProcessAction string

const (
	ProcessRestart ProcessAction = "restart"
	ProcessStop    ProcessAction = "stop"
	ProcessStart   ProcessAction = "start"
)

// DeploySource describes how an application was last deployed, as reported
// by apps:report, e.g. Type "git-sync" with Metadata "<repo>#<sha>".
type DeploySource struct {
	Type     string
	Metadata string
}

// RepoURL returns the repository of a git:sync deployment, or "" for other
// deploy sources.
func (d DeploySource) RepoURL() string {
	if d.Type != "git-sync" || d.Metadata == "" {
		return ""
	}
	if i := strings.LastIndex(d.Metadata, "#"); i > 0 {
		return d.Metadata[:i]
	}
	return d.Metadata
}
