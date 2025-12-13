package audit

import (
	"context"
	"time"
)

type Event struct {
	Timestamp    time.Time
	TenantID     string
	UserID       string
	Action       string
	Resource     string
	Parameters   map[string]interface{}
	Result       string
	ErrorMessage string
	Duration     time.Duration
	RequestID    string
	Metadata     map[string]string
}

type EventSink interface {
	Record(ctx context.Context, event Event) error
	Close() error
}

type NoOpSink struct{}

func NewNoOpSink() *NoOpSink {
	return &NoOpSink{}
}

func (s *NoOpSink) Record(ctx context.Context, event Event) error {
	return nil
}

func (s *NoOpSink) Close() error {
	return nil
}
