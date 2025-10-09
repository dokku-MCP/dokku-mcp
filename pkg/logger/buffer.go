package logger

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"time"
)

// RingBuffer stores recent log lines in-memory with a fixed capacity.
type RingBuffer struct {
	mu       sync.RWMutex
	entries  []string
	capacity int
	start    int
	count    int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 1000
	}
	return &RingBuffer{capacity: capacity, entries: make([]string, capacity)}
}

func (b *RingBuffer) Append(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.capacity == 0 {
		return
	}
	if b.count < b.capacity {
		idx := (b.start + b.count) % b.capacity
		b.entries[idx] = line
		b.count++
		return
	}
	b.entries[b.start] = line
	b.start = (b.start + 1) % b.capacity
}

func (b *RingBuffer) GetLast(n int) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.count == 0 {
		return []string{}
	}
	if n <= 0 || n > b.count {
		n = b.count
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		idx := (b.start + b.count - n + i) % b.capacity
		out[i] = b.entries[idx]
	}
	return out
}

// Capacity returns maximum number of entries the buffer can hold
func (b *RingBuffer) Capacity() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.capacity
}

// Size returns the current number of stored entries
func (b *RingBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

// bufferingHandler tees records to an underlying handler and also writes to ring buffer.
type bufferingHandler struct {
	next   slog.Handler
	buffer *RingBuffer
	opts   slog.HandlerOptions
}

func newBufferingHandler(next slog.Handler, buffer *RingBuffer, opts *slog.HandlerOptions) slog.Handler {
	var o slog.HandlerOptions
	if opts != nil {
		o = *opts
	}
	return &bufferingHandler{next: next, buffer: buffer, opts: o}
}

func (h *bufferingHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.next.Enabled(ctx, lvl)
}

func (h *bufferingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Format a simple line for buffer (RFC3339 time, level, message, attrs)
	var buf bytes.Buffer
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	buf.WriteString(ts.Format(time.RFC3339))
	buf.WriteString(" ")
	buf.WriteString(r.Level.String())
	buf.WriteString(" ")
	buf.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(a.Value.String())
		return true
	})
	h.buffer.Append(buf.String())
	return h.next.Handle(ctx, r)
}

func (h *bufferingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bufferingHandler{next: h.next.WithAttrs(attrs), buffer: h.buffer, opts: h.opts}
}

func (h *bufferingHandler) WithGroup(name string) slog.Handler {
	return &bufferingHandler{next: h.next.WithGroup(name), buffer: h.buffer, opts: h.opts}
}
