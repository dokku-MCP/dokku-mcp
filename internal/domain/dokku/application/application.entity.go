package application

import (
	"fmt"
	"time"
)

// ProcessType represents a type of process (web, worker, etc.)
type ProcessType string

const (
	ProcessTypeWeb    ProcessType = "web"
	ProcessTypeWorker ProcessType = "worker"
	ProcessTypeCron   ProcessType = "cron"
)

// BuildpackName represents a validated buildpack name
type BuildpackName struct {
	value string
}

func NewBuildpackName(name string) (*BuildpackName, error) {
	if name == "" {
		return nil, fmt.Errorf("buildpack name cannot be empty")
	}
	return &BuildpackName{value: name}, nil
}

func (b *BuildpackName) Value() string {
	return b.value
}

func (b *BuildpackName) Equal(other *BuildpackName) bool {
	if other == nil {
		return false
	}
	return b.value == other.value
}

// DomainName represents a validated domain name
type DomainName struct {
	value string
}

func NewDomainName(name string) (*DomainName, error) {
	if name == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}
	// Basic domain validation - could be enhanced
	return &DomainName{value: name}, nil
}

func (d *DomainName) Value() string {
	return d.value
}

func (d *DomainName) Equal(other *DomainName) bool {
	if other == nil {
		return false
	}
	return d.value == other.value
}

// GitRef represents a Git reference (branch, tag, commit)
type GitRef struct {
	value string
}

func NewGitRef(ref string) (*GitRef, error) {
	if ref == "" {
		return nil, fmt.Errorf("git reference cannot be empty")
	}
	return &GitRef{value: ref}, nil
}

func (g *GitRef) Value() string {
	return g.value
}

type Application struct {
	name *ApplicationName

	state     *ApplicationState
	createdAt time.Time
	updatedAt time.Time

	configuration *ApplicationConfiguration

	deploymentInfo *DeploymentInfo

	events []DomainEvent
}

type ApplicationConfiguration struct {
	buildpack       *BuildpackName
	domains         []*DomainName
	environmentVars map[string]string
	processes       map[ProcessType]*Process
}

type DeploymentInfo struct {
	currentGitRef   *GitRef
	lastDeployedAt  *time.Time
	buildImage      string
	runImage        string
	deploymentCount int
}

type Process struct {
	processType ProcessType
	command     string
	scale       int
}

type DomainEvent interface {
	OccurredAt() time.Time
	EventType() string
	AggregateID() string
}

func NewApplication(name string) (*Application, error) {
	appName, err := NewApplicationName(name)
	if err != nil {
		return nil, fmt.Errorf("impossible de créer l'application: %w", err)
	}

	initialState, err := NewApplicationState(StateCreated)
	if err != nil {
		return nil, fmt.Errorf("impossible d'initialiser l'état: %w", err)
	}

	app := &Application{
		name:      appName,
		state:     initialState,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		configuration: &ApplicationConfiguration{
			domains:         make([]*DomainName, 0),
			environmentVars: make(map[string]string),
			processes:       make(map[ProcessType]*Process),
		},
		deploymentInfo: &DeploymentInfo{
			deploymentCount: 0,
		},
		events: make([]DomainEvent, 0),
	}

	// Publier événement de création
	app.addEvent(NewApplicationCreatedEvent(appName.Value(), time.Now()))

	return app, nil
}

func (a *Application) Name() *ApplicationName { return a.name }

func (a *Application) State() *ApplicationState { return a.state }

func (a *Application) CreatedAt() time.Time { return a.createdAt }

func (a *Application) UpdatedAt() time.Time { return a.updatedAt }

func (a *Application) Configuration() *ApplicationConfiguration {
	return a.copyConfiguration()
}

func (a *Application) Deploy(gitRef *GitRef, buildOpts *DeploymentOptions) error {
	if gitRef == nil {
		return fmt.Errorf("la Git reference can't be null")
	}

	if !a.canDeploy() {
		return fmt.Errorf("the application can't be deployed in the state %s", a.state.Value())
	}

	if err := a.changeState(StateDeploying); err != nil {
		return fmt.Errorf("impossible to change the state for the deployment: %w", err)
	}

	a.deploymentInfo.currentGitRef = gitRef
	now := time.Now()
	a.deploymentInfo.lastDeployedAt = &now
	a.deploymentInfo.deploymentCount++

	if buildOpts != nil {
		a.deploymentInfo.buildImage = buildOpts.BuildImage
		a.deploymentInfo.runImage = buildOpts.RunImage
	}

	a.updatedAt = time.Now()
	a.addEvent(NewApplicationDeployedEvent(a.name.Value(), gitRef.Value(), time.Now()))

	return nil
}

func (a *Application) CompleteDeployment() error {
	if a.state.Value() != StateDeploying {
		return fmt.Errorf("no deployment in progress")
	}

	return a.changeState(StateRunning)
}

func (a *Application) FailDeployment(reason string) error {
	if a.state.Value() != StateDeploying {
		return fmt.Errorf("no deployment in progress")
	}
	a.addEvent(NewApplicationDeploymentFailedEvent(a.name.Value(), reason, time.Now()))

	return a.changeState(StateError)
}

func (a *Application) Scale(processType ProcessType, instances int) error {
	if instances < 0 {
		return fmt.Errorf("the number of instances can't be negative")
	}

	process, exists := a.configuration.processes[processType]
	if !exists {
		return fmt.Errorf("the process %s doesn't exist", processType)
	}

	oldScale := process.scale
	process.scale = instances
	a.updatedAt = time.Now()
	a.addEvent(NewApplicationScaledEvent(a.name.Value(), string(processType), oldScale, instances, time.Now()))

	return nil
}

func (a *Application) AddDomain(domainName string) error {
	domainVO, err := NewDomainName(domainName)
	if err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}

	for _, existingDomain := range a.configuration.domains {
		if existingDomain.Equal(domainVO) {
			return fmt.Errorf("the domain %s already exists", domainName)
		}
	}

	a.configuration.domains = append(a.configuration.domains, domainVO)
	a.updatedAt = time.Now()
	a.addEvent(NewDomainAddedEvent(a.name.Value(), domainName, time.Now()))

	return nil
}

func (a *Application) RemoveDomain(domainName string) error {
	domainVO, err := NewDomainName(domainName)
	if err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}

	for i, existingDomain := range a.configuration.domains {
		if existingDomain.Equal(domainVO) {
			// Delete the domain
			a.configuration.domains = append(a.configuration.domains[:i], a.configuration.domains[i+1:]...)
			a.updatedAt = time.Now()
			a.addEvent(NewDomainRemovedEvent(a.name.Value(), domainName, time.Now()))
			return nil
		}
	}

	return fmt.Errorf("the domain %s doesn't exist", domainName)
}

func (a *Application) SetBuildpack(buildpackName string) error {
	if buildpackName == "" {
		a.configuration.buildpack = nil
		a.updatedAt = time.Now()
		return nil
	}

	buildpackVO, err := NewBuildpackName(buildpackName)
	if err != nil {
		return fmt.Errorf("invalid buildpack: %w", err)
	}

	a.configuration.buildpack = buildpackVO
	a.updatedAt = time.Now()
	a.addEvent(NewBuildpackChangedEvent(a.name.Value(), buildpackName, time.Now()))

	return nil
}

func (a *Application) SetEnvironmentVariable(key, value string) error {
	if key == "" {
		return fmt.Errorf("the environment variable key can't be empty")
	}

	a.configuration.environmentVars[key] = value
	a.updatedAt = time.Now()

	return nil
}

func (a *Application) AddProcess(processType ProcessType, command string, scale int) error {
	if command == "" {
		return fmt.Errorf("the process command can't be empty")
	}

	if scale < 0 {
		return fmt.Errorf("the number of instances can't be negative")
	}

	process := &Process{
		processType: processType,
		command:     command,
		scale:       scale,
	}

	a.configuration.processes[processType] = process
	a.updatedAt = time.Now()

	return nil
}

func (a *Application) IsRunning() bool {
	return a.state.Value() == StateRunning
}

func (a *Application) IsDeployed() bool {
	currentState := a.state.Value()
	return currentState == StateRunning ||
		currentState == StateStopped ||
		currentState == StateDeployed
}

func (a *Application) HasDomain(domainName string) bool {
	domainVO, err := NewDomainName(domainName)
	if err != nil {
		return false
	}

	for _, existingDomain := range a.configuration.domains {
		if existingDomain.Equal(domainVO) {
			return true
		}
	}
	return false
}

func (a *Application) GetProcessScale(processType ProcessType) int {
	if process, exists := a.configuration.processes[processType]; exists {
		return process.scale
	}
	return 0
}

func (a *Application) GetDomains() []string {
	domains := make([]string, len(a.configuration.domains))
	for i, domainVO := range a.configuration.domains {
		domains[i] = domainVO.Value()
	}
	return domains
}

func (a *Application) GetEvents() []DomainEvent {
	return a.events
}

func (a *Application) ClearEvents() {
	a.events = make([]DomainEvent, 0)
}

// Private methods

func (a *Application) canDeploy() bool {
	currentState := a.state.Value()
	return currentState == StateCreated ||
		currentState == StateRunning ||
		currentState == StateStopped ||
		currentState == StateError
}

func (a *Application) changeState(newState StateValue) error {
	newStateVO, err := NewApplicationState(newState)
	if err != nil {
		return err
	}

	if !a.state.CanTransitionTo(newStateVO) {
		return fmt.Errorf("invalid state transition from %s to %s",
			a.state.Value(), newState)
	}

	oldState := a.state.Value()
	a.state = newStateVO
	a.updatedAt = time.Now()
	a.addEvent(NewApplicationStateChangedEvent(a.name.Value(),
		string(oldState), string(newState), time.Now()))

	return nil
}

func (a *Application) addEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

func (a *Application) copyConfiguration() *ApplicationConfiguration {
	domains := make([]*DomainName, len(a.configuration.domains))
	copy(domains, a.configuration.domains)

	envVars := make(map[string]string)
	for k, v := range a.configuration.environmentVars {
		envVars[k] = v
	}

	processes := make(map[ProcessType]*Process)
	for k, v := range a.configuration.processes {
		processes[k] = &Process{
			processType: v.processType,
			command:     v.command,
			scale:       v.scale,
		}
	}

	return &ApplicationConfiguration{
		buildpack:       a.configuration.buildpack,
		domains:         domains,
		environmentVars: envVars,
		processes:       processes,
	}
}

type DeploymentOptions struct {
	BuildImage string
	RunImage   string
	ForceClean bool
	NoCache    bool
}
