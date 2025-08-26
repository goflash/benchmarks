package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/goflash/benchmarks/internal/progress"
	"github.com/goflash/benchmarks/internal/types"
)

// ProcessState represents the state of a framework process
type ProcessState int

const (
	ProcessStateStopped ProcessState = iota
	ProcessStateStarting
	ProcessStateRunning
	ProcessStateFailed
	ProcessStateRestarting
)

func (s ProcessState) String() string {
	switch s {
	case ProcessStateStopped:
		return "stopped"
	case ProcessStateStarting:
		return "starting"
	case ProcessStateRunning:
		return "running"
	case ProcessStateFailed:
		return "failed"
	case ProcessStateRestarting:
		return "restarting"
	default:
		return "unknown"
	}
}

// ManagedProcess represents a managed framework process
type ManagedProcess struct {
	Framework    types.Framework
	Process      *os.Process
	Cmd          *exec.Cmd
	State        ProcessState
	StartTime    time.Time
	RestartCount int
	LastError    error
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// ProcessManager manages framework processes
type ProcessManager struct {
	processes    map[string]*ManagedProcess
	config       *types.Config
	tracker      *progress.Tracker
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	shuttingDown bool
}

// RestartPolicy defines when and how processes should be restarted
type RestartPolicy struct {
	MaxRestarts         int           // Maximum number of restart attempts
	RestartDelay        time.Duration // Delay between restart attempts
	BackoffMultiplier   float64       // Exponential backoff multiplier
	MaxRestartDelay     time.Duration // Maximum restart delay
	HealthCheckInterval time.Duration // How often to check process health
}

// DefaultRestartPolicy returns a sensible default restart policy
func DefaultRestartPolicy() RestartPolicy {
	return RestartPolicy{
		MaxRestarts:         10,
		RestartDelay:        5 * time.Second,
		BackoffMultiplier:   1.5,
		MaxRestartDelay:     60 * time.Second,
		HealthCheckInterval: 5 * time.Second,
	}
}

// RestartPolicyFromConfig creates a restart policy from configuration
func RestartPolicyFromConfig(config *types.Config) RestartPolicy {
	return RestartPolicy{
		MaxRestarts:         config.Process.MaxRestarts,
		RestartDelay:        time.Duration(config.Process.RestartDelay) * time.Second,
		BackoffMultiplier:   config.Process.BackoffMultiplier,
		MaxRestartDelay:     time.Duration(config.Process.MaxRestartDelay) * time.Second,
		HealthCheckInterval: time.Duration(config.Process.HealthCheckInterval) * time.Second,
	}
}

// NewProcessManager creates a new process manager
func NewProcessManager(config *types.Config, tracker *progress.Tracker) *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessManager{
		processes: make(map[string]*ManagedProcess),
		config:    config,
		tracker:   tracker,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// StartFramework starts a framework process and monitors it
func (pm *ProcessManager) StartFramework(frameworkName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	framework, exists := pm.config.Frameworks[frameworkName]
	if !exists {
		return fmt.Errorf("framework %s not found in configuration", frameworkName)
	}

	// Check if already running
	if proc, exists := pm.processes[frameworkName]; exists {
		proc.mu.RLock()
		state := proc.State
		proc.mu.RUnlock()

		if state == ProcessStateRunning || state == ProcessStateStarting {
			return fmt.Errorf("framework %s is already %s", frameworkName, state.String())
		}
	}

	// Create managed process
	ctx, cancel := context.WithCancel(pm.ctx)
	managedProc := &ManagedProcess{
		Framework: framework,
		State:     ProcessStateStarting,
		ctx:       ctx,
		cancel:    cancel,
	}

	pm.processes[frameworkName] = managedProc

	// Start the process
	if err := pm.startProcess(managedProc); err != nil {
		managedProc.mu.Lock()
		managedProc.State = ProcessStateFailed
		managedProc.LastError = err
		managedProc.mu.Unlock()
		return fmt.Errorf("failed to start %s: %w", frameworkName, err)
	}

	// Start monitoring
	pm.wg.Add(1)
	go pm.monitorProcess(frameworkName, managedProc)

	pm.tracker.LogInfo("Started framework %s on port %d", framework.Name, framework.Port)
	return nil
}

// startProcess starts the actual OS process
func (pm *ProcessManager) startProcess(managedProc *ManagedProcess) error {
	framework := managedProc.Framework
	binaryPath := filepath.Join("build", framework.BinaryName)

	// Ensure we have absolute path to prevent race conditions
	if absPath, err := filepath.Abs(binaryPath); err == nil {
		binaryPath = absPath
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", binaryPath)
	}

	// Create command
	cmd := exec.CommandContext(managedProc.ctx, binaryPath)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", framework.Port))

	// Set process group to allow proper cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	managedProc.mu.Lock()
	managedProc.Cmd = cmd
	managedProc.Process = cmd.Process
	managedProc.State = ProcessStateRunning
	managedProc.StartTime = time.Now()
	managedProc.mu.Unlock()

	return nil
}

// monitorProcess monitors a process and handles restarts
func (pm *ProcessManager) monitorProcess(frameworkName string, managedProc *ManagedProcess) {
	defer pm.wg.Done()

	policy := RestartPolicyFromConfig(pm.config)

	for {
		select {
		case <-managedProc.ctx.Done():
			return
		case <-pm.ctx.Done():
			return
		default:
			// Wait for process to exit
			managedProc.mu.RLock()
			cmd := managedProc.Cmd
			managedProc.mu.RUnlock()

			if cmd == nil {
				time.Sleep(policy.HealthCheckInterval)
				continue
			}

			// Wait for process to exit
			err := cmd.Wait()

			// Check if we're shutting down before attempting restart
			pm.mu.RLock()
			isShuttingDown := pm.shuttingDown
			pm.mu.RUnlock()
			
			if isShuttingDown {
				return
			}
			
			select {
			case <-managedProc.ctx.Done():
				return
			case <-pm.ctx.Done():
				return
			default:
			}

			managedProc.mu.Lock()
			// Only update state if we're not already in a restart process
			if managedProc.State != ProcessStateRestarting {
				managedProc.State = ProcessStateFailed
				managedProc.LastError = err
			}
			managedProc.mu.Unlock()

			pm.tracker.LogWarning("Framework %s process exited: %v", frameworkName, err)

			// Check if we should restart
			if managedProc.RestartCount >= policy.MaxRestarts {
				pm.tracker.LogError("Framework %s exceeded maximum restart attempts (%d)", frameworkName, policy.MaxRestarts)
				return
			}

			// Calculate restart delay with exponential backoff
			delay := time.Duration(float64(policy.RestartDelay) *
				(1 + float64(managedProc.RestartCount)*policy.BackoffMultiplier))
			if delay > policy.MaxRestartDelay {
				delay = policy.MaxRestartDelay
			}

			pm.tracker.LogInfo("Restarting framework %s in %v (attempt %d/%d)",
				frameworkName, delay, managedProc.RestartCount+1, policy.MaxRestarts)

			// Wait before restart
			select {
			case <-time.After(delay):
			case <-managedProc.ctx.Done():
				return
			case <-pm.ctx.Done():
				return
			}

			// Attempt restart
			managedProc.mu.Lock()
			managedProc.State = ProcessStateRestarting
			managedProc.RestartCount++
			managedProc.mu.Unlock()

			if err := pm.startProcess(managedProc); err != nil {
				pm.tracker.LogError("Failed to restart framework %s: %v", frameworkName, err)
				managedProc.mu.Lock()
				managedProc.State = ProcessStateFailed
				managedProc.LastError = err
				managedProc.mu.Unlock()

				// Continue to try again after delay
				continue
			}

			pm.tracker.LogSuccess("Successfully restarted framework %s", frameworkName)
		}
	}
}

// StopFramework stops a framework process
func (pm *ProcessManager) StopFramework(frameworkName string) error {
	pm.mu.RLock()
	managedProc, exists := pm.processes[frameworkName]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	managedProc.mu.Lock()
	defer managedProc.mu.Unlock()

	if managedProc.State == ProcessStateStopped {
		return nil
	}

	// Cancel context to stop monitoring
	managedProc.cancel()

	// Kill process if running
	if managedProc.Process != nil {
		// Try graceful shutdown first
		if err := managedProc.Process.Signal(syscall.SIGTERM); err != nil {
			pm.tracker.LogWarning("Failed to send SIGTERM to %s: %v", frameworkName, err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- managedProc.Cmd.Wait()
		}()

		select {
		case <-time.After(10 * time.Second):
			// Force kill if not shutdown gracefully
			pm.tracker.LogWarning("Force killing framework %s after timeout", frameworkName)
			if err := managedProc.Process.Kill(); err != nil {
				pm.tracker.LogError("Failed to kill %s: %v", frameworkName, err)
			}
		case <-done:
			// Process exited gracefully
		}
	}

	managedProc.State = ProcessStateStopped
	pm.tracker.LogInfo("Stopped framework %s", frameworkName)
	return nil
}

// StartAllFrameworks starts all configured frameworks
func (pm *ProcessManager) StartAllFrameworks() error {
	pm.tracker.LogInfo("Starting all framework processes...")

	var errors []error
	var wg sync.WaitGroup

	for frameworkName := range pm.config.Frameworks {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := pm.StartFramework(name); err != nil {
				errors = append(errors, fmt.Errorf("failed to start %s: %w", name, err))
			}
		}(frameworkName)
	}

	wg.Wait()

	if len(errors) > 0 {
		for _, err := range errors {
			pm.tracker.LogError("%v", err)
		}
		return fmt.Errorf("failed to start %d frameworks", len(errors))
	}

	pm.tracker.LogSuccess("All frameworks started successfully")
	return nil
}

// StopAllFrameworks stops all managed frameworks
func (pm *ProcessManager) StopAllFrameworks() error {
	pm.tracker.LogInfo("Stopping all framework processes...")

	pm.mu.RLock()
	frameworks := make([]string, 0, len(pm.processes))
	for name := range pm.processes {
		frameworks = append(frameworks, name)
	}
	pm.mu.RUnlock()

	var wg sync.WaitGroup
	for _, name := range frameworks {
		wg.Add(1)
		go func(frameworkName string) {
			defer wg.Done()
			if err := pm.StopFramework(frameworkName); err != nil {
				pm.tracker.LogError("Failed to stop %s: %v", frameworkName, err)
			}
		}(name)
	}

	wg.Wait()
	pm.tracker.LogSuccess("All frameworks stopped")
	return nil
}

// WaitForHealthy waits for all frameworks to be healthy
func (pm *ProcessManager) WaitForHealthy(ctx context.Context) error {
	pm.tracker.LogInfo("Waiting for all frameworks to be healthy...")

	var wg sync.WaitGroup
	errors := make(chan error, len(pm.config.Frameworks))

	for frameworkName, framework := range pm.config.Frameworks {
		wg.Add(1)
		go func(name string, fw types.Framework) {
			defer wg.Done()
			if err := pm.waitForFrameworkHealthy(ctx, fw); err != nil {
				errors <- fmt.Errorf("framework %s health check failed: %w", name, err)
			}
		}(frameworkName, framework)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var healthErrors []error
	for err := range errors {
		healthErrors = append(healthErrors, err)
	}

	if len(healthErrors) > 0 {
		for _, err := range healthErrors {
			pm.tracker.LogError("%v", err)
		}
		return fmt.Errorf("health check failed for %d frameworks", len(healthErrors))
	}

	pm.tracker.LogSuccess("All frameworks are healthy")
	return nil
}

// waitForFrameworkHealthy waits for a specific framework to be healthy
func (pm *ProcessManager) waitForFrameworkHealthy(ctx context.Context, framework types.Framework) error {
	interval := time.Duration(pm.config.Benchmark.HealthCheckInterval * float64(time.Second))
	timeout := time.Duration(pm.config.Benchmark.HealthCheckTimeout) * time.Second

	healthURL := fmt.Sprintf("%s/ping", framework.URL)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Try to connect to the server
			cmd := exec.CommandContext(ctx, "curl", "-f", "-s", "--max-time", "5", healthURL)
			if err := cmd.Run(); err == nil {
				return nil
			}

			time.Sleep(interval)
		}
	}
}

// GetFrameworkStatus returns the status of a framework
func (pm *ProcessManager) GetFrameworkStatus(frameworkName string) (ProcessState, error) {
	pm.mu.RLock()
	managedProc, exists := pm.processes[frameworkName]
	pm.mu.RUnlock()

	if !exists {
		return ProcessStateStopped, fmt.Errorf("framework %s not found", frameworkName)
	}

	managedProc.mu.RLock()
	defer managedProc.mu.RUnlock()
	return managedProc.State, nil
}

// GetAllStatuses returns the status of all frameworks
func (pm *ProcessManager) GetAllStatuses() map[string]ProcessState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	statuses := make(map[string]ProcessState)
	for name, proc := range pm.processes {
		proc.mu.RLock()
		statuses[name] = proc.State
		proc.mu.RUnlock()
	}
	return statuses
}

// IsFrameworkHealthy checks if a framework is running and healthy
func (pm *ProcessManager) IsFrameworkHealthy(frameworkName string) bool {
	framework, exists := pm.config.Frameworks[frameworkName]
	if !exists {
		return false
	}

	// Check process state
	state, err := pm.GetFrameworkStatus(frameworkName)
	if err != nil || (state != ProcessStateRunning && state != ProcessStateStarting) {
		return false
	}

	// Quick health check with shorter timeout for better responsiveness
	healthURL := fmt.Sprintf("%s/ping", framework.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-f", "-s", "--max-time", "2", "--connect-timeout", "1", healthURL)
	return cmd.Run() == nil
}

// EnsureFrameworkRunning ensures a framework is running, restarting if necessary
func (pm *ProcessManager) EnsureFrameworkRunning(frameworkName string) error {
	// First quick health check
	if pm.IsFrameworkHealthy(frameworkName) {
		return nil
	}

	pm.tracker.LogWarning("Framework %s is not healthy, attempting to restart", frameworkName)

	pm.mu.Lock()
	managedProc, exists := pm.processes[frameworkName]
	pm.mu.Unlock()

	if !exists {
		// Framework not managed yet, start it fresh
		return pm.StartFramework(frameworkName)
	}

	managedProc.mu.Lock()
	currentState := managedProc.State
	managedProc.mu.Unlock()

	// If already restarting, wait for it to complete
	if currentState == ProcessStateRestarting {
		pm.tracker.LogInfo("Framework %s is already restarting, waiting...", frameworkName)
		// Wait for restart to complete
		for i := 0; i < 30; i++ { // Wait up to 30 seconds
			time.Sleep(1 * time.Second)
			if pm.IsFrameworkHealthy(frameworkName) {
				return nil
			}
		}
		return fmt.Errorf("framework %s restart timed out", frameworkName)
	}

	// Force restart by stopping and starting
	pm.tracker.LogInfo("Force restarting framework %s", frameworkName)

	// Stop the current process
	if err := pm.stopProcessOnly(frameworkName); err != nil {
		pm.tracker.LogWarning("Failed to stop %s before restart: %v", frameworkName, err)
	}

	// Wait a moment for cleanup
	time.Sleep(3 * time.Second)

	// Start fresh
	if err := pm.startProcessForRestart(frameworkName); err != nil {
		return fmt.Errorf("failed to restart %s: %w", frameworkName, err)
	}

	// Wait for health with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	framework := pm.config.Frameworks[frameworkName]
	if err := pm.waitForFrameworkHealthy(ctx, framework); err != nil {
		return fmt.Errorf("framework %s failed health check after restart: %w", frameworkName, err)
	}

	pm.tracker.LogSuccess("Framework %s successfully restarted and healthy", frameworkName)
	return nil
}

// stopProcessOnly stops just the process without affecting monitoring
func (pm *ProcessManager) stopProcessOnly(frameworkName string) error {
	pm.mu.RLock()
	managedProc, exists := pm.processes[frameworkName]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	managedProc.mu.Lock()
	defer managedProc.mu.Unlock()

	if managedProc.Process != nil {
		// Try graceful shutdown first
		if err := managedProc.Process.Signal(syscall.SIGTERM); err != nil {
			pm.tracker.LogWarning("Failed to send SIGTERM to %s: %v", frameworkName, err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			if managedProc.Cmd != nil && managedProc.Cmd.ProcessState == nil {
				// Only wait if process hasn't already exited
				done <- managedProc.Cmd.Wait()
			} else {
				done <- nil
			}
		}()

		select {
		case <-time.After(5 * time.Second):
			// Force kill if not shutdown gracefully
			pm.tracker.LogWarning("Force killing framework %s after timeout", frameworkName)
			if err := managedProc.Process.Kill(); err != nil {
				pm.tracker.LogError("Failed to kill %s: %v", frameworkName, err)
			}
		case <-done:
			// Process exited
		}
	}

	managedProc.State = ProcessStateFailed
	return nil
}

// startProcessForRestart starts a process specifically for restart scenarios
func (pm *ProcessManager) startProcessForRestart(frameworkName string) error {
	pm.mu.RLock()
	managedProc, exists := pm.processes[frameworkName]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("framework %s not found", frameworkName)
	}

	// Create new context for the restarted process
	ctx, cancel := context.WithCancel(pm.ctx)
	managedProc.mu.Lock()
	managedProc.ctx = ctx
	managedProc.cancel = cancel
	managedProc.State = ProcessStateStarting
	managedProc.mu.Unlock()

	// Start the process
	if err := pm.startProcess(managedProc); err != nil {
		managedProc.mu.Lock()
		managedProc.State = ProcessStateFailed
		managedProc.LastError = err
		managedProc.mu.Unlock()
		return fmt.Errorf("failed to start process: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the process manager
func (pm *ProcessManager) Shutdown() error {
	pm.tracker.LogInfo("Shutting down process manager...")

	// Set shutdown flag to prevent restarts
	pm.mu.Lock()
	pm.shuttingDown = true
	pm.mu.Unlock()

	// Stop all frameworks first
	if err := pm.StopAllFrameworks(); err != nil {
		pm.tracker.LogError("Error during framework shutdown: %v", err)
	}

	// Cancel context to stop all monitoring
	pm.cancel()

	// Wait for all monitors to finish with timeout
	done := make(chan struct{})
	go func() {
		pm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-time.After(10 * time.Second):
		pm.tracker.LogWarning("Timeout waiting for monitoring goroutines to finish")
	}

	pm.tracker.LogSuccess("Process manager shutdown complete")
	return nil
}
