package application

import (
	"time"
)

type ApplicationCreatedEvent struct {
	aggregateID string
	occurredAt  time.Time
}

func NewApplicationCreatedEvent(aggregateID string, occurredAt time.Time) *ApplicationCreatedEvent {
	return &ApplicationCreatedEvent{
		aggregateID: aggregateID,
		occurredAt:  occurredAt,
	}
}

func (e *ApplicationCreatedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *ApplicationCreatedEvent) EventType() string     { return "application.created" }
func (e *ApplicationCreatedEvent) AggregateID() string   { return e.aggregateID }

type ApplicationDeployedEvent struct {
	aggregateID string
	gitRef      string
	occurredAt  time.Time
}

func NewApplicationDeployedEvent(aggregateID, gitRef string, occurredAt time.Time) *ApplicationDeployedEvent {
	return &ApplicationDeployedEvent{
		aggregateID: aggregateID,
		gitRef:      gitRef,
		occurredAt:  occurredAt,
	}
}

func (e *ApplicationDeployedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *ApplicationDeployedEvent) EventType() string     { return "application.deployed" }
func (e *ApplicationDeployedEvent) AggregateID() string   { return e.aggregateID }
func (e *ApplicationDeployedEvent) GitRef() string        { return e.gitRef }

type ApplicationDeploymentFailedEvent struct {
	aggregateID string
	reason      string
	occurredAt  time.Time
}

func NewApplicationDeploymentFailedEvent(aggregateID, reason string, occurredAt time.Time) *ApplicationDeploymentFailedEvent {
	return &ApplicationDeploymentFailedEvent{
		aggregateID: aggregateID,
		reason:      reason,
		occurredAt:  occurredAt,
	}
}

func (e *ApplicationDeploymentFailedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *ApplicationDeploymentFailedEvent) EventType() string     { return "application.deployment.failed" }
func (e *ApplicationDeploymentFailedEvent) AggregateID() string   { return e.aggregateID }
func (e *ApplicationDeploymentFailedEvent) Reason() string        { return e.reason }

type ApplicationScaledEvent struct {
	aggregateID string
	processType string
	oldScale    int
	newScale    int
	occurredAt  time.Time
}

func NewApplicationScaledEvent(aggregateID, processType string, oldScale, newScale int, occurredAt time.Time) *ApplicationScaledEvent {
	return &ApplicationScaledEvent{
		aggregateID: aggregateID,
		processType: processType,
		oldScale:    oldScale,
		newScale:    newScale,
		occurredAt:  occurredAt,
	}
}

func (e *ApplicationScaledEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *ApplicationScaledEvent) EventType() string     { return "application.scaled" }
func (e *ApplicationScaledEvent) AggregateID() string   { return e.aggregateID }
func (e *ApplicationScaledEvent) ProcessType() string   { return e.processType }
func (e *ApplicationScaledEvent) OldScale() int         { return e.oldScale }
func (e *ApplicationScaledEvent) NewScale() int         { return e.newScale }

type ApplicationStateChangedEvent struct {
	aggregateID string
	oldState    string
	newState    string
	occurredAt  time.Time
}

func NewApplicationStateChangedEvent(aggregateID, oldState, newState string, occurredAt time.Time) *ApplicationStateChangedEvent {
	return &ApplicationStateChangedEvent{
		aggregateID: aggregateID,
		oldState:    oldState,
		newState:    newState,
		occurredAt:  occurredAt,
	}
}

func (e *ApplicationStateChangedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *ApplicationStateChangedEvent) EventType() string     { return "application.state.changed" }
func (e *ApplicationStateChangedEvent) AggregateID() string   { return e.aggregateID }
func (e *ApplicationStateChangedEvent) OldState() string      { return e.oldState }
func (e *ApplicationStateChangedEvent) NewState() string      { return e.newState }

type DomainAddedEvent struct {
	aggregateID string
	domain      string
	occurredAt  time.Time
}

func NewDomainAddedEvent(aggregateID, domain string, occurredAt time.Time) *DomainAddedEvent {
	return &DomainAddedEvent{
		aggregateID: aggregateID,
		domain:      domain,
		occurredAt:  occurredAt,
	}
}

func (e *DomainAddedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *DomainAddedEvent) EventType() string     { return "application.domain.added" }
func (e *DomainAddedEvent) AggregateID() string   { return e.aggregateID }
func (e *DomainAddedEvent) Domain() string        { return e.domain }

type DomainRemovedEvent struct {
	aggregateID string
	domain      string
	occurredAt  time.Time
}

func NewDomainRemovedEvent(aggregateID, domain string, occurredAt time.Time) *DomainRemovedEvent {
	return &DomainRemovedEvent{
		aggregateID: aggregateID,
		domain:      domain,
		occurredAt:  occurredAt,
	}
}

func (e *DomainRemovedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *DomainRemovedEvent) EventType() string     { return "application.domain.removed" }
func (e *DomainRemovedEvent) AggregateID() string   { return e.aggregateID }
func (e *DomainRemovedEvent) Domain() string        { return e.domain }

type BuildpackChangedEvent struct {
	aggregateID string
	buildpack   string
	occurredAt  time.Time
}

func NewBuildpackChangedEvent(aggregateID, buildpack string, occurredAt time.Time) *BuildpackChangedEvent {
	return &BuildpackChangedEvent{
		aggregateID: aggregateID,
		buildpack:   buildpack,
		occurredAt:  occurredAt,
	}
}

func (e *BuildpackChangedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *BuildpackChangedEvent) EventType() string     { return "application.buildpack.changed" }
func (e *BuildpackChangedEvent) AggregateID() string   { return e.aggregateID }
func (e *BuildpackChangedEvent) Buildpack() string     { return e.buildpack }
