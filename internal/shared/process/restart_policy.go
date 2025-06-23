package process

import (
	"fmt"
	"strconv"
	"strings"
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

// NewRestartPolicyFromString creates a RestartPolicy from a Dokku-compatible string.
func NewRestartPolicyFromString(policyStr string) (*RestartPolicy, error) {
	if policyStr == "" {
		return nil, fmt.Errorf("restart policy string cannot be empty")
	}

	if policyStr == "no" {
		policyStr = string(RestartPolicyNever)
	}

	parts := strings.SplitN(policyStr, ":", 2)
	policyType := RestartPolicyType(parts[0])

	if !isValidRestartPolicyType(policyType) {
		return nil, fmt.Errorf("invalid restart policy type: %s", policyType)
	}

	rp, err := NewRestartPolicy(policyType)
	if err != nil {
		return nil, err // Should not happen given the check above
	}

	if policyType == RestartPolicyOnFailure {
		// Default is set in NewRestartPolicy. Override if specified.
		if len(parts) == 2 {
			max, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid max restarts count in policy: %s", policyStr)
			}
			rp.WithMaxRestarts(max)
		}
	} else if len(parts) > 1 {
		return nil, fmt.Errorf("max restarts count is only applicable for 'on-failure' policy: %s", policyStr)
	}

	return rp, nil
}

func NewRestartPolicy(policyType RestartPolicyType) (*RestartPolicy, error) {
	if !isValidRestartPolicyType(policyType) {
		return nil, fmt.Errorf("invalid restart policy type: %s", policyType)
	}

	return &RestartPolicy{
		policy:         policyType,
		maxRestarts:    10,
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
