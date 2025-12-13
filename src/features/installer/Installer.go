package installer

import (
	"context"
	"fmt"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

// Installer manages the system installation process
type Installer struct {
	distro  Distro
	steps   []*InstallStep
	logger  *utility.Logger
	shell   *utility.Shell
	useTUI  bool
	dryRun  bool
}

// NewInstaller creates a new installer instance
func NewInstaller(logger *utility.Logger, useTUI bool) (*Installer, error) {
	// Detect distribution
	distro, err := DetectDistro()
	if err != nil {
		return nil, fmt.Errorf("failed to detect distribution: %w", err)
	}

	if !IsSupported(distro) {
		return nil, fmt.Errorf("distribution '%s' is not supported yet", distro)
	}

	shell := utility.NewShell(logger)

	installer := &Installer{
		distro: distro,
		logger: logger,
		shell:  shell,
		useTUI: useTUI,
		dryRun: false,
	}

	// Initialize steps based on distro
	installer.initializeSteps()

	return installer, nil
}

// initializeSteps sets up the installation steps based on distro
func (i *Installer) initializeSteps() {
	switch i.distro {
	case Arch:
		i.steps = i.getArchSteps()
	case Fedora:
		i.steps = i.getFedoraSteps()
	case Debian, Ubuntu:
		i.steps = i.getDebianSteps()
	default:
		i.steps = []*InstallStep{}
	}
}

// Run executes all installation steps
func (i *Installer) Run(ctx context.Context) error {
	i.logger.Info("===========================================")
	i.logger.Info("  Daemira Installer")
	i.logger.Info("  Distribution: %s", i.distro)
	i.logger.Info("  Steps: %d", len(i.steps))
	i.logger.Info("===========================================")
	i.logger.Info("")

	startTime := time.Now()
	var failedSteps []*InstallStep
	var skippedSteps []*InstallStep
	var successSteps []*InstallStep

	// Execute each step
	for idx, step := range i.steps {
		i.logger.Info("Step %d/%d: %s", idx+1, len(i.steps), step.Name)

		if err := step.Run(ctx, i); err != nil {
			failedSteps = append(failedSteps, step)

			// Ask user if they want to continue on error
			i.logger.Warn("Step failed. Continue with remaining steps? (y/N)")
			// For now, continue automatically
			// In TUI mode, this would be interactive
			continue
		}

		if step.Status == Skipped {
			skippedSteps = append(skippedSteps, step)
		} else if step.Status == Success {
			successSteps = append(successSteps, step)
		}

		i.logger.Info("")
	}

	duration := time.Since(startTime)

	// Print summary
	i.logger.Info("")
	i.logger.Info("===========================================")
	i.logger.Info("  Installation Summary")
	i.logger.Info("===========================================")
	i.logger.Info("Duration: %v", duration)
	i.logger.Info("Total Steps: %d", len(i.steps))
	i.logger.Info("✓ Successful: %d", len(successSteps))
	i.logger.Info("⊘ Skipped: %d", len(skippedSteps))
	i.logger.Info("✗ Failed: %d", len(failedSteps))

	if len(failedSteps) > 0 {
		i.logger.Error("")
		i.logger.Error("Failed Steps:")
		for _, step := range failedSteps {
			i.logger.Error("  - %s", step.Summary())
		}
		return fmt.Errorf("%d steps failed", len(failedSteps))
	}

	i.logger.Info("")
	i.logger.Info("===========================================")
	i.logger.Info("  Installation Complete!")
	i.logger.Info("===========================================")
	i.logger.Info("")
	i.logger.Info("Next steps:")
	i.logger.Info("  1. Reboot your system to apply all changes")
	i.logger.Info("  2. Log in to Hyprland")
	i.logger.Info("  3. Run 'daemira status' to check system status")
	i.logger.Info("")

	return nil
}

// RunStep executes a specific step by ID
func (i *Installer) RunStep(ctx context.Context, stepID string) error {
	for _, step := range i.steps {
		if step.ID == stepID {
			return step.Run(ctx, i)
		}
	}
	return fmt.Errorf("step '%s' not found", stepID)
}

// ListSteps returns all installation steps
func (i *Installer) ListSteps() []*InstallStep {
	return i.steps
}

// GetDistro returns the detected distribution
func (i *Installer) GetDistro() Distro {
	return i.distro
}
