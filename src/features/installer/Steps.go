package installer

import (
	"context"
	"fmt"
)

// StepStatus represents the status of an installation step
type StepStatus int

const (
	Pending StepStatus = iota
	Running
	Success
	Warning
	Failed
	Skipped
)

func (s StepStatus) String() string {
	switch s {
	case Pending:
		return "Pending"
	case Running:
		return "Running"
	case Success:
		return "Success"
	case Warning:
		return "Warning"
	case Failed:
		return "Failed"
	case Skipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// Icon returns a visual indicator for the step status
func (s StepStatus) Icon() string {
	switch s {
	case Pending:
		return "⏳"
	case Running:
		return "⟳"
	case Success:
		return "✓"
	case Warning:
		return "⚠"
	case Failed:
		return "✗"
	case Skipped:
		return "⊘"
	default:
		return "?"
	}
}

// InstallStep represents a single installation step
type InstallStep struct {
	ID          string
	Name        string
	Description string
	Status      StepStatus
	Error       error
	Execute     func(ctx context.Context, installer *Installer) error
	Skip        func(installer *Installer) bool
}

// NewInstallStep creates a new installation step
func NewInstallStep(id, name, description string, execute func(ctx context.Context, installer *Installer) error) *InstallStep {
	return &InstallStep{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      Pending,
		Execute:     execute,
		Skip:        func(i *Installer) bool { return false },
	}
}

// Run executes the installation step
func (s *InstallStep) Run(ctx context.Context, installer *Installer) error {
	// Check if step should be skipped
	if s.Skip != nil && s.Skip(installer) {
		s.Status = Skipped
		installer.logger.Info("[%s] %s - Skipped", s.Status.Icon(), s.Name)
		return nil
	}

	// Mark as running
	s.Status = Running
	installer.logger.Info("[%s] %s - %s", s.Status.Icon(), s.Name, s.Description)

	// Execute the step
	err := s.Execute(ctx, installer)
	if err != nil {
		s.Status = Failed
		s.Error = err
		installer.logger.Error("[%s] %s - Failed: %v", s.Status.Icon(), s.Name, err)
		return fmt.Errorf("step '%s' failed: %w", s.ID, err)
	}

	// Mark as successful
	s.Status = Success
	installer.logger.Info("[%s] %s - Complete", s.Status.Icon(), s.Name)
	return nil
}

// Summary returns a summary string for the step
func (s *InstallStep) Summary() string {
	if s.Error != nil {
		return fmt.Sprintf("[%s] %s - %s (Error: %v)", s.Status.Icon(), s.Name, s.Status.String(), s.Error)
	}
	return fmt.Sprintf("[%s] %s - %s", s.Status.Icon(), s.Name, s.Status.String())
}
