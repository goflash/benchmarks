package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goflash/benchmarks/internal/types"
)

// Tracker handles progress tracking and display
type Tracker struct {
	verbose     bool
	progressDir string
	stateFile   string
}

// ProgressState represents the current progress state
type ProgressState struct {
	RunID              string             `json:"run_id"`
	StartTime          time.Time          `json:"start_time"`
	LastUpdate         time.Time          `json:"last_update"`
	CurrentFramework   string             `json:"current_framework"`
	CurrentScenario    string             `json:"current_scenario"`
	CurrentBatch       int                `json:"current_batch"`
	TotalBatches       int                `json:"total_batches"`
	CurrentRetry       int                `json:"current_retry"`
	MaxRetries         int                `json:"max_retries"`
	CompletedTests     int                `json:"completed_tests"`
	TotalTests         int                `json:"total_tests"`
	CompletedTestsList []string           `json:"completed_tests_list"`
	FailedTests        []string           `json:"failed_tests"`
	Results            []types.TestResult `json:"results"`
	Config             types.Config       `json:"config"`
	Status             string             `json:"status"`
}

// NewTracker creates a new progress tracker
func NewTracker(verbose bool) *Tracker {
	return &Tracker{
		verbose:     verbose,
		progressDir: "",
		stateFile:   "",
	}
}

// SetResultsDir sets the results directory for progress tracking
func (t *Tracker) SetResultsDir(resultsDir string) {
	t.progressDir = resultsDir
	t.stateFile = filepath.Join(resultsDir, "progress_state.json")
}

// LoadState loads the current progress state from JSON
func (t *Tracker) LoadState() (*ProgressState, error) {
	if _, err := os.Stat(t.stateFile); os.IsNotExist(err) {
		return nil, nil // No state file exists
	}

	data, err := os.ReadFile(t.stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ProgressState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SaveState saves the current progress state to JSON
func (t *Tracker) SaveState(state *ProgressState) error {
	// Ensure progress directory exists
	if err := os.MkdirAll(t.progressDir, 0755); err != nil {
		return fmt.Errorf("failed to create progress directory: %w", err)
	}

	state.LastUpdate = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(t.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// ClearState removes the current progress state
func (t *Tracker) ClearState() error {
	if _, err := os.Stat(t.stateFile); os.IsNotExist(err) {
		return nil // File doesn't exist
	}

	return os.Remove(t.stateFile)
}

// InitializeProgress initializes progress tracking
func (t *Tracker) InitializeProgress(config *types.Config) {
	// Check if there's an existing incomplete state to resume from
	existingState, err := t.LoadState()
	if err == nil && existingState != nil && existingState.Status != "completed" {
		// Resume from existing state - preserve completed tests
		t.LogInfo("Resuming from existing progress state with %d completed tests", len(existingState.CompletedTestsList))

		// Update the existing state with new config if needed
		existingState.Config = *config
		existingState.TotalTests = len(config.Frameworks) * len(config.Scenarios) * config.Benchmark.Batches
		existingState.TotalBatches = config.Benchmark.Batches
		existingState.MaxRetries = config.Benchmark.MaxRetries
		existingState.LastUpdate = time.Now()
		existingState.Status = "running"

		if err := t.SaveState(existingState); err != nil {
			t.LogWarning("Failed to save resumed progress state: %v", err)
		}
		return
	}

	// Also check if there's a completed state with tests that we can resume from
	// This handles the case where someone runs a subset of tests, then runs the full suite
	if err == nil && existingState != nil && existingState.Status == "completed" && len(existingState.CompletedTestsList) > 0 {
		// Resume from completed state - preserve completed tests but start fresh run
		t.LogInfo("Resuming from completed state with %d existing tests, starting fresh run", len(existingState.CompletedTestsList))

		// Create new state but preserve the completed tests list
		state := &ProgressState{
			RunID:              fmt.Sprintf("run_%s", time.Now().Format("20060102_150405")),
			StartTime:          time.Now(),
			LastUpdate:         time.Now(),
			TotalBatches:       config.Benchmark.Batches,
			MaxRetries:         config.Benchmark.MaxRetries,
			TotalTests:         len(config.Frameworks) * len(config.Scenarios) * config.Benchmark.Batches,
			CompletedTestsList: existingState.CompletedTestsList, // Preserve completed tests
			FailedTests:        make([]string, 0),
			Results:            existingState.Results, // Preserve existing results
			Config:             *config,
			Status:             "running",
		}

		if err := t.SaveState(state); err != nil {
			t.LogWarning("Failed to save resumed progress state: %v", err)
		}
		return
	}

	// Create new state for fresh run
	state := &ProgressState{
		RunID:              fmt.Sprintf("run_%s", time.Now().Format("20060102_150405")),
		StartTime:          time.Now(),
		LastUpdate:         time.Now(),
		TotalBatches:       config.Benchmark.Batches,
		MaxRetries:         config.Benchmark.MaxRetries,
		TotalTests:         len(config.Frameworks) * len(config.Scenarios) * config.Benchmark.Batches,
		CompletedTestsList: make([]string, 0),
		FailedTests:        make([]string, 0),
		Results:            make([]types.TestResult, 0),
		Config:             *config,
		Status:             "running",
	}

	if err := t.SaveState(state); err != nil {
		t.LogWarning("Failed to save initial progress state: %v", err)
	}
}

// UpdateProgress updates the progress state (lightweight - no save)
func (t *Tracker) UpdateProgress(framework, scenario string, batch, retry, completedTests int) {
	// This method now only updates display, not state file
	// State file is saved atomically when test completes
}

// AddResult adds a test result to the progress state and saves atomically
func (t *Tracker) AddResult(result types.TestResult) {
	state, err := t.LoadState()
	if err != nil {
		t.LogWarning("Failed to load progress state: %v", err)
		return
	}

	if state == nil {
		return
	}

	// Update state with completed test
	state.Results = append(state.Results, result)
	state.CurrentFramework = result.Framework
	state.CurrentScenario = result.Scenario
	state.CurrentBatch = result.Batch
	state.CompletedTests = len(state.Results)

	// Add to completed tests list atomically
	testKey := fmt.Sprintf("%s_%s_%d", result.Framework, result.Scenario, result.Batch)
	found := false
	for _, completed := range state.CompletedTestsList {
		if completed == testKey {
			found = true
			break
		}
	}
	if !found {
		state.CompletedTestsList = append(state.CompletedTestsList, testKey)
	}

	// Save state atomically when test completes
	if err := t.SaveState(state); err != nil {
		t.LogWarning("Failed to save progress state: %v", err)
	}
}

// MarkTestFailed marks a test as failed and saves atomically
func (t *Tracker) MarkTestFailed(framework, scenario string, batch int, error string) {
	state, err := t.LoadState()
	if err != nil {
		t.LogWarning("Failed to load progress state: %v", err)
		return
	}

	if state == nil {
		return
	}

	failedTest := fmt.Sprintf("%s_%s_%d: %s", framework, scenario, batch, error)
	state.FailedTests = append(state.FailedTests, failedTest)
	state.CurrentFramework = framework
	state.CurrentScenario = scenario
	state.CurrentBatch = batch

	// Save state atomically when test fails
	if err := t.SaveState(state); err != nil {
		t.LogWarning("Failed to save progress state: %v", err)
	}
}

// GetResumeInfo returns information needed to resume from a failed run
func (t *Tracker) GetResumeInfo() (*types.ResumeInfo, error) {
	state, err := t.LoadState()
	if err != nil {
		return nil, err
	}

	if state == nil {
		return nil, nil // No state to resume from
	}

	return &types.ResumeInfo{
		RunID:          state.RunID,
		LastFramework:  state.CurrentFramework,
		LastScenario:   state.CurrentScenario,
		LastBatch:      state.CurrentBatch,
		LastRetry:      state.CurrentRetry,
		CompletedTests: state.CompletedTestsList,
		FailedTests:    state.FailedTests,
		ResultsDir:     t.progressDir,
		Config:         state.Config,
	}, nil
}

// PrintHeader prints the benchmark header
func (t *Tracker) PrintHeader() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🚀 Go Web Framework Benchmark Suite")
	fmt.Println(strings.Repeat("=", 80))
}

// PrintConfig prints the configuration summary
func (t *Tracker) PrintConfig(config *types.Config) {
	fmt.Printf("📊 Frameworks: %d\n", len(config.Frameworks))
	fmt.Printf("🧪 Scenarios: %d\n", len(config.Scenarios))
	fmt.Printf("🔄 Batches: %d\n", config.Benchmark.Batches)
	fmt.Printf("🛠️  Tool: %s\n", config.Benchmark.Tool)
	fmt.Printf("📈 Requests: %d\n", config.Benchmark.DefaultRequests)
	fmt.Printf("🔗 Connections: %d\n", config.Benchmark.DefaultConnections)
	fmt.Println(strings.Repeat("-", 80))
}

// UpdateFramework updates the current framework progress
func (t *Tracker) UpdateFramework(current, total int, framework string) {
	if t.verbose {
		fmt.Printf("\n🏗️  Framework %d/%d: %s\n", current, total, framework)
	}
}

// UpdateScenario updates the current scenario progress
func (t *Tracker) UpdateScenario(current, total int, scenario string) {
	if t.verbose {
		fmt.Printf("  📝 Scenario %d/%d: %s\n", current, total, scenario)
	}
}

// UpdateBatch updates the current batch progress
func (t *Tracker) UpdateBatch(current, total, batch int) {
	if t.verbose {
		fmt.Printf("    🔄 Batch %d/%d\n", current, total)
	}
}

// UpdateOverall updates the overall progress
func (t *Tracker) UpdateOverall(completed, total int) {
	percentage := float64(completed) / float64(total) * 100
	fmt.Printf("📊 Overall Progress: %d/%d (%.1f%%)\n", completed, total, percentage)
}

// UpdateDetailedProgress updates progress with more granular information
func (t *Tracker) UpdateDetailedProgress(completed, total int, currentFramework, currentScenario string, batch, totalBatches int) {
	percentage := float64(completed) / float64(total) * 100
	fmt.Printf("📊 Progress: %d/%d (%.1f%%) - [%s] %s (Batch %d/%d)\n",
		completed, total, percentage, currentFramework, currentScenario, batch, totalBatches)
}

// LogInfo logs an info message
func (t *Tracker) LogInfo(format string, args ...interface{}) {
	// Clear the current line and move to a new line
	fmt.Print("\r\033[K") // Clear current line
	fmt.Printf("ℹ️  "+format+"\n", args...)
}

// LogSuccess logs a success message
func (t *Tracker) LogSuccess(format string, args ...interface{}) {
	fmt.Printf("✅ "+format+"\n", args...)
}

// LogWarning logs a warning message
func (t *Tracker) LogWarning(format string, args ...interface{}) {
	// Clear the current line and move to a new line
	fmt.Print("\r\033[K") // Clear current line
	fmt.Printf("⚠️  "+format+"\n", args...)
}

// LogError logs an error message
func (t *Tracker) LogError(format string, args ...interface{}) {
	// Clear the current line and move to a new line
	fmt.Print("\r\033[K") // Clear current line
	fmt.Printf("❌ "+format+"\n", args...)
}

// LogTestResult logs a test result
func (t *Tracker) LogTestResult(result *types.TestResult) {
	if t.verbose {
		fmt.Printf("    📊 %s: %.0f RPS (%.2fms)\n",
			result.Scenario, result.RequestsPerSec,
			float64(result.LatencyMean.Microseconds())/1000)
	} else {
		// Always show test results, even in non-verbose mode
		fmt.Printf("✓ [%s] %s: %.0f RPS (%.2fms)\n",
			result.Framework, result.Scenario, result.RequestsPerSec,
			float64(result.LatencyMean.Microseconds())/1000)
	}
}

// LogCurrentTest logs the current test being executed
func (t *Tracker) LogCurrentTest(framework, scenario string, batch, totalBatches int) {
	fmt.Printf("🧪 Testing: [%s] %s (Batch %d/%d)\n", framework, scenario, batch, totalBatches)
}

// PrintSummary prints the benchmark summary
func (t *Tracker) PrintSummary(run *types.TestRun) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("🆔 Run ID: %s\n", run.ID)
	fmt.Printf("⏱️  Duration: %s\n", run.Duration)
	fmt.Printf("📈 Total Tests: %d\n", len(run.Results))
	fmt.Printf("✅ Status: %s\n", run.Status)

	if run.ErrorMessage != "" {
		fmt.Printf("❌ Error: %s\n", run.ErrorMessage)
	}

	fmt.Println(strings.Repeat("=", 80))
}

// Finish finalizes the progress tracking
func (t *Tracker) Finish() {
	// Mark the final state as completed instead of clearing it
	state, err := t.LoadState()
	if err == nil && state != nil {
		// Only mark as completed if all tests are actually completed
		if state.CompletedTests >= state.TotalTests {
			state.Status = "completed"
			if err := t.SaveState(state); err != nil {
				t.LogWarning("Failed to save final progress state: %v", err)
			}
		} else {
			// Keep status as "running" if not all tests are completed
			t.LogInfo("Benchmark run completed but not all tests finished (%d/%d). State kept as 'running' for resume.",
				state.CompletedTests, state.TotalTests)
		}
	}
	fmt.Println("🎉 Benchmark completed!")
}
