package utility

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Shell provides command execution capabilities
type Shell struct {
	logger *Logger
}

// Result contains the output of a command execution
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
	Duration time.Duration
	Command  string
}

// ExecOptions configures command execution
type ExecOptions struct {
	Timeout        time.Duration
	StdoutCallback func(line string)
	StderrCallback func(line string)
	Env            map[string]string
	WorkDir        string
	UseSudo        bool
}

// NewShell creates a new Shell executor
func NewShell(logger *Logger) *Shell {
	return &Shell{logger: logger}
}

// Execute runs a command with the given options
func (s *Shell) Execute(ctx context.Context, command string, opts *ExecOptions) (*Result, error) {
	if opts == nil {
		opts = &ExecOptions{
			Timeout: 30 * time.Second,
		}
	}

	// Set default timeout if not specified
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	// Add sudo if requested
	if opts.UseSudo {
		command = fmt.Sprintf("sudo %s", command)
	}

	// Create context with timeout
	execCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	startTime := time.Now()

	// Create command
	cmd := exec.CommandContext(execCtx, "bash", "-c", command)

	// Set working directory
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Set environment variables
	if len(opts.Env) > 0 {
		cmd.Env = append(cmd.Env, s.envMapToSlice(opts.Env)...)
	}

	// Create stdout and stderr pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Capture stdout
	var stdoutBuf bytes.Buffer
	stdoutDone := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutBuf.WriteString(line + "\n")
			if opts.StdoutCallback != nil {
				opts.StdoutCallback(line)
			}
		}
		close(stdoutDone)
	}()

	// Capture stderr
	var stderrBuf bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuf.WriteString(line + "\n")
			if opts.StderrCallback != nil {
				opts.StderrCallback(line)
			}
		}
		close(stderrDone)
	}()

	// Wait for output reading to complete
	<-stdoutDone
	<-stderrDone

	// Wait for command to complete
	err = cmd.Wait()
	duration := time.Since(startTime)

	result := &Result{
		ExitCode: 0,
		Stdout:   strings.TrimSpace(stdoutBuf.String()),
		Stderr:   strings.TrimSpace(stderrBuf.String()),
		TimedOut: false,
		Duration: duration,
		Command:  command,
	}

	// Check if command timed out
	if execCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = -1
		return result, fmt.Errorf("command timed out after %v", opts.Timeout)
	}

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			return result, fmt.Errorf("command failed: %w", err)
		}
	}

	return result, nil
}

// envMapToSlice converts a map of environment variables to a slice
func (s *Shell) envMapToSlice(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for key, value := range env {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}

// QuickExec is a convenience method for simple command execution
func (s *Shell) QuickExec(command string) (*Result, error) {
	return s.Execute(context.Background(), command, nil)
}

// ExecWithTimeout runs a command with a specific timeout
func (s *Shell) ExecWithTimeout(command string, timeout time.Duration) (*Result, error) {
	return s.Execute(context.Background(), command, &ExecOptions{Timeout: timeout})
}

// ExecWithSudo runs a command with sudo
func (s *Shell) ExecWithSudo(command string) (*Result, error) {
	return s.Execute(context.Background(), command, &ExecOptions{UseSudo: true})
}
