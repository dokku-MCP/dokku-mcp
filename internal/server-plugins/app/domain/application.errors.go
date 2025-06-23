package app

import "errors"

// Application domain specific errors
var (
	ErrApplicationNotFound      = errors.New("application not found")
	ErrRepositoryUnavailable    = errors.New("repository unavailable")
	ErrApplicationAlreadyExists = errors.New("application already exists")
	ErrInvalidApplicationName   = errors.New("invalid application name")
	ErrApplicationNotDeployed   = errors.New("application not deployed")
	ErrDeploymentInProgress     = errors.New("deployment already in progress")
	ErrInvalidState             = errors.New("invalid application state")
)
