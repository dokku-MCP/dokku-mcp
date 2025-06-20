package domain

import "errors"

var (
	ErrDeploymentNotFound       = errors.New("deployment not found")
	ErrDeploymentNotRunning     = errors.New("deployment is not running")
	ErrDeploymentAlreadyRunning = errors.New("deployment is already running")
	ErrInvalidDeploymentStatus  = errors.New("invalid deployment status")
	ErrDeploymentAlreadyExists  = errors.New("deployment already exists")
)
