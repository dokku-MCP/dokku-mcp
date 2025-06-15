package application

import "errors"

var (
	ErrApplicationNotFound      = errors.New("application not found")
	ErrRepositoryUnavailable    = errors.New("repository unavailable")
	ErrApplicationAlreadyExists = errors.New("application already exists")
	ErrInvalidApplicationName   = errors.New("invalid application name")
)
