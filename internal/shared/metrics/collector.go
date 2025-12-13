package metrics

import (
	"context"
	"time"
)

type Collector interface {
	RecordToolExecution(ctx context.Context, toolName string, duration time.Duration, success bool)
	RecordDokkuCommand(ctx context.Context, command string, duration time.Duration, success bool)
	RecordTenantActivity(ctx context.Context, tenantID string)
	RecordAuthenticationAttempt(ctx context.Context, success bool)
	RecordAuthorizationCheck(ctx context.Context, resource, action string, allowed bool)
	Close() error
}

type NoOpCollector struct{}

func NewNoOpCollector() *NoOpCollector {
	return &NoOpCollector{}
}

func (c *NoOpCollector) RecordToolExecution(ctx context.Context, toolName string, duration time.Duration, success bool) {
}

func (c *NoOpCollector) RecordDokkuCommand(ctx context.Context, command string, duration time.Duration, success bool) {
}

func (c *NoOpCollector) RecordTenantActivity(ctx context.Context, tenantID string) {
}

func (c *NoOpCollector) RecordAuthenticationAttempt(ctx context.Context, success bool) {
}

func (c *NoOpCollector) RecordAuthorizationCheck(ctx context.Context, resource, action string, allowed bool) {
}

func (c *NoOpCollector) Close() error {
	return nil
}
