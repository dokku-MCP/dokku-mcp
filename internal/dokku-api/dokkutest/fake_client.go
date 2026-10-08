// Package dokkutest provides an in-memory DokkuClient for tests.
package dokkutest

import (
	"context"
	"fmt"
	"strings"
	"sync"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
)

// Call records one command executed through the fake client.
type Call struct {
	Command string
	Args    []string
	Stdin   string
}

// String renders the call the way it would be sent over SSH.
func (c Call) String() string {
	return strings.TrimSpace(c.Command + " " + strings.Join(c.Args, " "))
}

// Handler produces the output of a command.
type Handler func(args []string) ([]byte, error)

// FakeClient is a DokkuClient that records every call and answers from
// per-command handlers. Unhandled commands succeed with empty output.
// Arguments are validated with the production rules, so a test fails if
// code under test builds an argument the real client would reject.
type FakeClient struct {
	mu       sync.Mutex
	handlers map[string]Handler
	calls    []Call
}

var _ dokkuApi.DokkuClient = (*FakeClient)(nil)

// NewFakeClient returns a fake client with no handlers.
func NewFakeClient() *FakeClient {
	return &FakeClient{handlers: make(map[string]Handler)}
}

// On registers the handler for a command and returns the client for chaining.
func (f *FakeClient) On(command string, handler Handler) *FakeClient {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[command] = handler
	return f
}

// Respond registers a fixed output for a command.
func (f *FakeClient) Respond(command, output string) *FakeClient {
	return f.On(command, func([]string) ([]byte, error) { return []byte(output), nil })
}

// Fail registers a fixed error for a command.
func (f *FakeClient) Fail(command string, err error) *FakeClient {
	return f.On(command, func([]string) ([]byte, error) { return nil, err })
}

// Calls returns a copy of the recorded calls.
func (f *FakeClient) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Call(nil), f.calls...)
}

// CallsTo returns the recorded calls for one command.
func (f *FakeClient) CallsTo(command string) []Call {
	var out []Call
	for _, c := range f.Calls() {
		if c.Command == command {
			out = append(out, c)
		}
	}
	return out
}

func (f *FakeClient) execute(command string, args []string, stdin string) ([]byte, error) {
	if err := f.ValidateCommand(command, args); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}
	f.mu.Lock()
	f.calls = append(f.calls, Call{Command: command, Args: append([]string(nil), args...), Stdin: stdin})
	handler := f.handlers[command]
	f.mu.Unlock()
	if handler == nil {
		return nil, nil
	}
	return handler(args)
}

func (f *FakeClient) ExecuteCommand(_ context.Context, command string, args []string) ([]byte, error) {
	return f.execute(command, args, "")
}

func (f *FakeClient) ExecuteCommandWithStdin(_ context.Context, command string, args []string, stdin string) ([]byte, error) {
	return f.execute(command, args, stdin)
}

// StreamLogs replays the output of the "logs" handler line by line, as if
// following `logs <app> -t`, then ends the stream.
func (f *FakeClient) StreamLogs(ctx context.Context, appName string) (<-chan dokkuApi.LogLine, <-chan error, error) {
	out, err := f.execute("logs", []string{appName, "-t"}, "")
	if err != nil {
		return nil, nil, err
	}
	lines := make(chan dokkuApi.LogLine)
	errs := make(chan error, 1)
	go func() {
		defer close(lines)
		defer close(errs)
		for line := range strings.Lines(string(out)) {
			select {
			case lines <- dokkuApi.LogLine{Message: strings.TrimRight(line, "\r\n")}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return lines, errs, nil
}

func (f *FakeClient) GetKeyValueOutput(ctx context.Context, command string, args []string, separator string) (map[string]string, error) {
	out, err := f.ExecuteCommand(ctx, command, args)
	if err != nil {
		return nil, err
	}
	return dokkuApi.ParseKeyValueOutput(string(out), separator), nil
}

func (f *FakeClient) GetListOutput(ctx context.Context, command string, args []string) ([]string, error) {
	out, err := f.ExecuteCommand(ctx, command, args)
	if err != nil {
		return nil, err
	}
	return dokkuApi.ParseListOutput(string(out), true), nil
}

func (f *FakeClient) GetTableOutput(ctx context.Context, command string, args []string, skipHeaders bool) ([]map[string]string, error) {
	out, err := f.ExecuteCommand(ctx, command, args)
	if err != nil {
		return nil, err
	}
	return dokkuApi.ParseTableOutput(string(out), skipHeaders), nil
}

func (f *FakeClient) ExecuteStructured(ctx context.Context, spec dokkuApi.CommandSpec) (*dokkuApi.CommandResult, error) {
	out, err := f.ExecuteCommand(ctx, spec.Command, spec.Args)
	if err != nil {
		return nil, err
	}
	return &dokkuApi.CommandResult{RawOutput: out}, nil
}

func (f *FakeClient) ExecuteWithAutoFormat(ctx context.Context, command string, args []string) (*dokkuApi.CommandResult, error) {
	return f.ExecuteStructured(ctx, dokkuApi.CommandSpec{Command: command, Args: args})
}

func (f *FakeClient) DiscoverCapabilities(context.Context) error { return nil }

func (f *FakeClient) GetCapabilities() *dokkuApi.DokkuCapabilities {
	return dokkuApi.NewDokkuCapabilities()
}

func (f *FakeClient) GetSSHConnectionManager() *dokkuApi.SSHConnectionManager { return nil }

func (f *FakeClient) SetBlacklist([]string) {}

func (f *FakeClient) SetAllowlist([]string) {}

func (f *FakeClient) ValidateCommand(command string, args []string) error {
	if command == "" {
		return fmt.Errorf("command name cannot be empty")
	}
	for i, arg := range args {
		if err := dokkuApi.ValidateArg(arg); err != nil {
			return fmt.Errorf("argument %d: %w", i, err)
		}
	}
	return nil
}
