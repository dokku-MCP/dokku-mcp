package process

import "fmt"

// Process represents a Dokku process.
type Process struct {
	processType ProcessType
	command     *ProcessCommand
	scale       *ProcessScale
}

// NewProcess creates a new Process.
func NewProcess(processType ProcessType, command string, scale int) (*Process, error) {
	cmd, err := NewProcessCommand(command)
	if err != nil {
		return nil, fmt.Errorf("invalid process command: %w", err)
	}

	ps, err := NewProcessScale(scale)
	if err != nil {
		return nil, fmt.Errorf("invalid process scale: %w", err)
	}
	return &Process{
		processType: processType,
		command:     cmd,
		scale:       ps,
	}, nil
}

// NewProcessForScaling creates a new Process for scaling without requiring a command.
// This is used when scaling a process type that doesn't exist yet - the command will be
// determined from the Procfile during deployment.
func NewProcessForScaling(processType ProcessType, scale int) (*Process, error) {
	ps, err := NewProcessScale(scale)
	if err != nil {
		return nil, fmt.Errorf("invalid process scale: %w", err)
	}
	return &Process{
		processType: processType,
		command:     nil, // Command will be determined later
		scale:       ps,
	}, nil
}

// Type returns the process type.
func (p *Process) Type() ProcessType {
	return p.processType
}

// Command returns the process command.
func (p *Process) Command() *ProcessCommand {
	return p.command
}

// HasCommand returns true if the process has a command defined.
func (p *Process) HasCommand() bool {
	return p.command != nil
}

// Scale returns the process scale.
func (p *Process) Scale() int {
	return p.scale.Value()
}

// SetScale updates the process scale.
func (p *Process) SetScale(scale int) error {
	ps, err := NewProcessScale(scale)
	if err != nil {
		return err
	}
	p.scale = ps
	return nil
}

// SetCommand updates the process command.
func (p *Process) SetCommand(command string) error {
	cmd, err := NewProcessCommand(command)
	if err != nil {
		return err
	}
	p.command = cmd
	return nil
}
