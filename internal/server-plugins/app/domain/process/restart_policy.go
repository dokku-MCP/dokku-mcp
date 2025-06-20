package process

import (
	"fmt"
	"time"
)

type RestartPolicy struct {
	policy         RestartPolicyType
	maxRestarts    int
	restartDelay   time.Duration
	backoffFactor  float64
	maxRestartTime time.Duration
}

type RestartPolicyType string

const (
	RestartPolicyAlways        RestartPolicyType = "always"
	RestartPolicyOnFailure     RestartPolicyType = "on-failure"
	RestartPolicyNever         RestartPolicyType = "never"
	RestartPolicyUnlessStopped RestartPolicyType = "unless-stopped"
)

func NewRestartPolicy(policyType RestartPolicyType) (*RestartPolicy, error) {
	if !isValidRestartPolicyType(policyType) {
		return nil, fmt.Errorf("invalid restart policy type: %s", policyType)
	}

	return &RestartPolicy{
		policy:         policyType,
		maxRestarts:    5,
		restartDelay:   10 * time.Second,
		backoffFactor:  2.0,
		maxRestartTime: 5 * time.Minute,
	}, nil
}

func (rp *RestartPolicy) WithMaxRestarts(max int) *RestartPolicy {
	if max < 0 {
		max = 0
	}
	rp.maxRestarts = max
	return rp
}

func (rp *RestartPolicy) WithRestartDelay(delay time.Duration) *RestartPolicy {
	if delay < 0 {
		delay = 0
	}
	rp.restartDelay = delay
	return rp
}

func (rp *RestartPolicy) Policy() RestartPolicyType {
	return rp.policy
}

func (rp *RestartPolicy) MaxRestarts() int {
	return rp.maxRestarts
}

func (rp *RestartPolicy) RestartDelay() time.Duration {
	return rp.restartDelay
}

func isValidRestartPolicyType(policyType RestartPolicyType) bool {
	validTypes := []RestartPolicyType{
		RestartPolicyAlways, RestartPolicyOnFailure,
		RestartPolicyNever, RestartPolicyUnlessStopped,
	}

	for _, validType := range validTypes {
		if policyType == validType {
			return true
		}
	}
	return false
}
