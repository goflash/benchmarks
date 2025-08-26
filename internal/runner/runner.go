package runner

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/goflash/benchmarks/internal/config"
	"github.com/goflash/benchmarks/internal/process"
	"github.com/goflash/benchmarks/internal/progress"
	"github.com/goflash/benchmarks/internal/types"
)

// Runner handles benchmark execution
type Runner struct {
	config         *types.Config
	tracker        *progress.Tracker
	processManager *process.ProcessManager
	results        []types.TestResult
	mu             sync.Mutex
}

// NewRunner creates a new benchmark runner
func NewRunner(config *types.Config, tracker *progress.Tracker) *Runner {
	return &Runner{
		config:         config,
		tracker:        tracker,
		processManager: process.NewProcessManager(config, tracker),
		results:        make([]types.TestResult, 0),
	}
}

// setResourceLimits configures system resource limits to prevent process killing
func (r *Runner) setResourceLimits() error {
	// Set file descriptor limit
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		r.tracker.LogWarning("Failed to get current file descriptor limit: %v", err)
	} else {
		// Set to the configured max or system max, whichever is smaller
		maxFD := uint64(r.config.System.MaxFileDescriptors)
		if maxFD > rLimit.Max {
			maxFD = rLimit.Max
		}

		rLimit.Cur = maxFD
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
			r.tracker.LogWarning("Failed to set file descriptor limit to %d: %v", maxFD, err)
		} else {
			r.tracker.LogInfo("Set file descriptor limit to %d", maxFD)
		}
	}

	return nil
}

// Run executes the complete benchmark suite
func (r *Runner) Run(ctx context.Context) (*types.TestRun, error) {
	// Set resource limits to prevent system killing processes
	if err := r.setResourceLimits(); err != nil {
		r.tracker.LogWarning("Failed to set resource limits: %v", err)
	}

	run := &types.TestRun{
		ID:        generateRunID(),
		StartTime: time.Now(),
		Status:    "running",
		Config:    *r.config,
	}

	r.tracker.PrintHeader()
	r.tracker.PrintConfig(r.config)

	// Create results directory
	resultsDir, err := r.createResultsDir()
	if err != nil {
		run.Status = "failed"
		run.ErrorMessage = fmt.Sprintf("failed to create results directory: %v", err)
		return run, err
	}

	// Set results directory for progress tracking
	r.tracker.SetResultsDir(resultsDir)

	// Check for resume BEFORE initializing progress
	resumeInfo, err := r.tracker.GetResumeInfo()
	if err != nil {
		r.tracker.LogWarning("Failed to get resume info: %v", err)
	}

	// Handle existing results based on progress state
	if err := r.handleExistingResults(resultsDir, resumeInfo); err != nil {
		r.tracker.LogWarning("Failed to handle existing results: %v", err)
	}

	// Initialize progress tracking (this will create new state or resume from existing)
	r.tracker.InitializeProgress(r.config)

	// Start framework processes
	if err := r.processManager.StartAllFrameworks(); err != nil {
		run.Status = "failed"
		run.ErrorMessage = fmt.Sprintf("failed to start frameworks: %v", err)
		return run, err
	}

	// Wait for frameworks to be healthy
	if err := r.processManager.WaitForHealthy(ctx); err != nil {
		run.Status = "failed"
		run.ErrorMessage = fmt.Sprintf("failed to wait for frameworks: %v", err)
		return run, err
	}

	// Run benchmarks
	if err := r.runBenchmarks(ctx, resultsDir, resumeInfo); err != nil {
		run.Status = "failed"
		run.ErrorMessage = fmt.Sprintf("failed to run benchmarks: %v", err)
		return run, err
	}

	// Save results
	if err := r.saveResults(resultsDir); err != nil {
		run.Status = "failed"
		run.ErrorMessage = fmt.Sprintf("failed to save results: %v", err)
		return run, err
	}

	// Generate README from template
	if err := r.generateREADME(resultsDir); err != nil {
		r.tracker.LogWarning("Failed to generate README: %v", err)
		// Don't fail the entire run if README generation fails
	}

	run.EndTime = time.Now()
	run.Duration = run.EndTime.Sub(run.StartTime)
	run.Status = "completed"
	run.Results = r.results

	r.tracker.PrintSummary(run)

	// Shutdown process manager
	if err := r.processManager.Shutdown(); err != nil {
		r.tracker.LogWarning("Error during process manager shutdown: %v", err)
	}

	r.tracker.Finish()

	return run, nil
}

// createResultsDir creates the results directory
func (r *Runner) createResultsDir() (string, error) {
	loader := config.NewLoader()
	_, err := loader.Load("")
	if err != nil {
		return "", err
	}
	return loader.CreateResultsDir()
}

// isRetryableError determines if an error is retryable (resource-related)
func (r *Runner) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for resource-related errors that should be retried
	retryablePatterns := []string{
		"signal: killed",
		"killed",
		"out of memory",
		"resource temporarily unavailable",
		"too many open files",
		"connection refused",
		"connection reset",
		"timeout",
		"context deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// logRetryStatistics logs statistics about retry attempts
func (r *Runner) logRetryStatistics(frameworkName, scenarioName string, retryCount int, totalRetries int) {
	if retryCount > 0 {
		r.tracker.LogInfo("Retry statistics for %s - %s: %d/%d attempts used", frameworkName, scenarioName, retryCount, totalRetries)
		if retryCount == totalRetries {
			r.tracker.LogWarning("Maximum retries reached for %s - %s", frameworkName, scenarioName)
		}
	}
}

// shouldSkipTest determines if a test should be skipped based on resume info
func (r *Runner) shouldSkipTest(frameworkName, scenarioName string, batch int, resumeInfo *types.ResumeInfo) bool {
	if resumeInfo == nil {
		return false
	}

	// Get framework display name from config to match the format used in AddResult
	frameworkDisplayName := frameworkName
	if framework, exists := r.config.Frameworks[frameworkName]; exists {
		frameworkDisplayName = framework.Name
	}

	// Get scenario display name from config to match the format used in AddResult
	scenarioDisplayName := scenarioName
	if scenario, exists := r.config.Scenarios[scenarioName]; exists {
		scenarioDisplayName = scenario.Name
	}

	testKey := fmt.Sprintf("%s_%s_%d", frameworkDisplayName, scenarioDisplayName, batch)
	for _, completed := range resumeInfo.CompletedTests {
		if completed == testKey {
			return true
		}
	}
	return false
}

// runBenchmarks runs all benchmark tests
func (r *Runner) runBenchmarks(ctx context.Context, resultsDir string, resumeInfo *types.ResumeInfo) error {
	r.tracker.LogInfo("Starting benchmark tests...")

	totalTests := len(r.config.Frameworks) * len(r.config.Scenarios) * r.config.Benchmark.Batches
	completedTests := 0
	lastProgressPercentage := 0.0
	var currentPercentage float64

	// Initialize completed test count for resume
	if resumeInfo != nil && len(resumeInfo.CompletedTests) > 0 {
		r.tracker.LogInfo("Resuming from previous run...")
		completedTests = len(resumeInfo.CompletedTests)
	}

	for frameworkName, framework := range r.config.Frameworks {
		r.tracker.UpdateFramework(completedTests/len(r.config.Scenarios)/r.config.Benchmark.Batches+1, len(r.config.Frameworks), frameworkName)

		for scenarioName, scenario := range r.config.Scenarios {
			r.tracker.UpdateScenario(completedTests/r.config.Benchmark.Batches%len(r.config.Scenarios)+1, len(r.config.Scenarios), scenarioName)

			for batch := 1; batch <= r.config.Benchmark.Batches; batch++ {
				r.tracker.UpdateBatch(batch, r.config.Benchmark.Batches, batch)

				// Show current test context
				r.tracker.LogCurrentTest(frameworkName, scenarioName, batch, r.config.Benchmark.Batches)

				// Update detailed progress every 1%
				currentPercentage = float64(completedTests) / float64(totalTests) * 100
				if currentPercentage >= lastProgressPercentage+1.0 || completedTests == 0 {
					r.tracker.UpdateDetailedProgress(completedTests, totalTests, frameworkName, scenarioName, batch, r.config.Benchmark.Batches)
					lastProgressPercentage = math.Floor(currentPercentage)
				}

				// Check if this test was already completed
				if r.shouldSkipTest(frameworkName, scenarioName, batch, resumeInfo) {
					completedTests++
					// Use display names for the log message to match the progress state format
					frameworkDisplayName := framework.Name
					scenarioDisplayName := scenario.Name
					r.tracker.LogInfo("Skipping already completed test: %s_%s_%d", frameworkDisplayName, scenarioDisplayName, batch)
					continue
				}

				// Run the test with framework restart capability
				result, err := r.runTestWithRestart(ctx, frameworkName, framework, scenario, batch, resultsDir)
				if err != nil {
					r.tracker.MarkTestFailed(frameworkName, scenarioName, batch, err.Error())
					return fmt.Errorf("test failed for %s - %s (Batch %d): %w", frameworkName, scenarioName, batch, err)
				}

				r.mu.Lock()
				r.results = append(r.results, *result)
				r.mu.Unlock()

				// Add result to progress tracking
				r.tracker.AddResult(*result)

				r.tracker.LogTestResult(result)
				completedTests++

				// Update detailed progress every 1% or every test
				currentPercentage = float64(completedTests) / float64(totalTests) * 100
				if currentPercentage >= lastProgressPercentage+1.0 || completedTests == totalTests {
					r.tracker.UpdateDetailedProgress(completedTests, totalTests, frameworkName, scenarioName, batch, r.config.Benchmark.Batches)
					lastProgressPercentage = math.Floor(currentPercentage)
				} else {
					r.tracker.UpdateOverall(completedTests, totalTests)
				}

				// Pause between batches
				if batch < r.config.Benchmark.Batches {
					time.Sleep(time.Duration(r.config.Benchmark.BatchPause) * time.Second)
				}
			}
		}
	}

	r.tracker.LogSuccess("All benchmark tests completed")
	return nil
}

// runTestWithRestart runs a test with automatic framework restart capability
func (r *Runner) runTestWithRestart(ctx context.Context, frameworkName string, framework types.Framework, scenario types.Scenario, batch int, resultsDir string) (*types.TestResult, error) {
	var result *types.TestResult
	var err error
	var retryCount int
	var frameworkRestarts int
	maxFrameworkRestarts := 3 // Maximum number of framework restarts per test

	for attempt := 0; attempt <= r.config.Benchmark.MaxRetries; attempt++ {
		// Ensure framework is running before each attempt
		if err := r.processManager.EnsureFrameworkRunning(frameworkName); err != nil {
			r.tracker.LogError("Framework %s is not available: %v", frameworkName, err)
			if frameworkRestarts < maxFrameworkRestarts {
				frameworkRestarts++
				r.tracker.LogWarning("Attempting framework restart %d/%d", frameworkRestarts, maxFrameworkRestarts)
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, fmt.Errorf("framework %s unavailable after %d restart attempts: %w", frameworkName, maxFrameworkRestarts, err)
		}

		// Run the actual test
		result, err = r.runTest(ctx, framework, scenario, batch, attempt, resultsDir)
		if err == nil {
			// Test succeeded
			if retryCount > 0 {
				r.tracker.LogSuccess("Test succeeded after %d retries", retryCount)
			}
			return result, nil
		}

		retryCount++
		r.tracker.LogWarning("Test attempt %d failed: %v", attempt+1, err)

		// Check if this is a retryable error
		if !r.isRetryableError(err) {
			r.tracker.LogError("Non-retryable error encountered: %v", err)
			break
		}

		// If we're not at the last attempt, prepare for retry
		if attempt < r.config.Benchmark.MaxRetries {
			// Check if framework is still healthy
			if !r.processManager.IsFrameworkHealthy(frameworkName) {
				r.tracker.LogWarning("Framework %s is not healthy after test failure", frameworkName)
				if frameworkRestarts < maxFrameworkRestarts {
					frameworkRestarts++
					r.tracker.LogInfo("Restarting framework %s (restart %d/%d)", frameworkName, frameworkRestarts, maxFrameworkRestarts)

					// Force restart the framework
					if restartErr := r.processManager.EnsureFrameworkRunning(frameworkName); restartErr != nil {
						r.tracker.LogError("Failed to restart framework %s: %v", frameworkName, restartErr)
						continue
					}

					r.tracker.LogSuccess("Framework %s restarted successfully", frameworkName)
				} else {
					r.tracker.LogError("Maximum framework restarts (%d) reached for %s", maxFrameworkRestarts, frameworkName)
					break
				}
			}

			// Exponential backoff with jitter
			baseSleep := time.Duration(r.config.Benchmark.RetrySleep) * time.Second
			exponentialSleep := baseSleep * time.Duration(1<<attempt) // 2^attempt
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			sleepTime := exponentialSleep + jitter

			r.tracker.LogInfo("Waiting %v before retry %d/%d (exponential backoff)", sleepTime, attempt+2, r.config.Benchmark.MaxRetries+1)
			time.Sleep(sleepTime)

			// Reset resource limits
			if retryErr := r.setResourceLimits(); retryErr != nil {
				r.tracker.LogWarning("Failed to reset resource limits before retry: %v", retryErr)
			}
		}
	}

	// Log final retry statistics
	r.logRetryStatistics(frameworkName, scenario.Name, retryCount-1, r.config.Benchmark.MaxRetries)

	return nil, fmt.Errorf("test failed after %d retries and %d framework restarts: %w", retryCount-1, frameworkRestarts, err)
}

// runTest runs a single benchmark test
func (r *Runner) runTest(ctx context.Context, framework types.Framework, scenario types.Scenario, batch, retry int, resultsDir string) (*types.TestResult, error) {
	// Prepare command based on benchmark tool
	var cmd *exec.Cmd
	var outputFile string

	// Sanitize scenario name for file path
	sanitizedName := strings.ReplaceAll(scenario.Name, " ", "_")
	sanitizedName = strings.ReplaceAll(sanitizedName, "/", "_")

	r.tracker.LogInfo("Original scenario name: %s", scenario.Name)
	r.tracker.LogInfo("Sanitized scenario name: %s", sanitizedName)

	// Create a fresh context with timeout for this individual test
	// Allow extra time beyond the test duration for framework startup and cleanup
	testDuration, _ := time.ParseDuration(r.config.Benchmark.DefaultDuration)
	testTimeout := testDuration + (60 * time.Second) // Add 60 seconds buffer
	testCtx, testCancel := context.WithTimeout(context.Background(), testTimeout)
	defer testCancel()

	if r.config.Benchmark.Tool == "wrk" {
		outputFile = filepath.Join(resultsDir, "raw", fmt.Sprintf("%s_%s_batch%d_retry%d.txt", framework.Name, sanitizedName, batch, retry))
		cmd = r.prepareWrkCommand(testCtx, framework, scenario, outputFile)
	} else {
		outputFile = filepath.Join(resultsDir, "raw", fmt.Sprintf("%s_%s_batch%d_retry%d.txt", framework.Name, sanitizedName, batch, retry))
		cmd = r.prepareAbCommand(testCtx, framework, scenario, outputFile)
	}

	// Ensure the raw directory exists before running the command
	rawDir := filepath.Join(resultsDir, "raw")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create raw directory: %w", err)
	}

	// Run the command with progress monitoring
	startTime := time.Now()

	// Start progress monitoring for long-running tests
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()

	go r.monitorTestProgress(monitorCtx, framework.Name, scenario.Name, batch, retry)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		r.tracker.LogWarning("Failed to create output directory: %v", err)
	}

	// Save raw output
	r.tracker.LogInfo("Saving output to: %s", outputFile)
	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		r.tracker.LogWarning("Failed to save raw output: %v", err)
	}

	if err != nil {
		return nil, fmt.Errorf("command failed: %w, output: %s", err, string(output))
	}

	// Parse results
	result, err := r.parseOutput(string(output), framework, scenario, batch, retry, duration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	return result, nil
}

// prepareWrkCommand prepares a wrk command
func (r *Runner) prepareWrkCommand(ctx context.Context, framework types.Framework, scenario types.Scenario, outputFile string) *exec.Cmd {
	args := []string{
		"-t", strconv.Itoa(r.config.Benchmark.Threads),
		"-c", strconv.Itoa(r.config.Benchmark.DefaultConnections),
	}

	// wrk only supports duration-based testing, not request count
	// If requests are specified, calculate an appropriate duration
	if r.config.Benchmark.DefaultRequests > 0 {
		// Estimate duration based on expected RPS (conservative estimate)
		estimatedRPS := 10000.0 // Conservative estimate
		estimatedDuration := float64(r.config.Benchmark.DefaultRequests) / estimatedRPS
		if estimatedDuration < 1.0 {
			estimatedDuration = 1.0 // Minimum 1 second
		}
		args = append(args, "-d", fmt.Sprintf("%.0fs", estimatedDuration))
	} else {
		args = append(args, "-d", r.config.Benchmark.DefaultDuration)
	}

	if r.config.Benchmark.KeepAlive {
		args = append(args, "-H", "Connection: keep-alive")
	}

	// For POST requests, use the lua script
	if scenario.Method == "POST" {
		args = append(args, "-s", "wrk/post.lua")
	}

	args = append(args, fmt.Sprintf("%s%s", framework.URL, scenario.Path))

	cmd := exec.CommandContext(ctx, "wrk", args...)
	return cmd
}

// prepareAbCommand prepares an ApacheBench command
func (r *Runner) prepareAbCommand(ctx context.Context, framework types.Framework, scenario types.Scenario, outputFile string) *exec.Cmd {
	args := []string{
		"-n", strconv.Itoa(r.config.Benchmark.DefaultRequests),
		"-c", strconv.Itoa(r.config.Benchmark.DefaultConnections),
	}

	if r.config.Benchmark.KeepAlive {
		args = append(args, "-k")
	}

	if scenario.Method == "POST" && scenario.BodyFile != "" {
		args = append(args, "-p", scenario.BodyFile)
	}

	args = append(args, fmt.Sprintf("%s%s", framework.URL, scenario.Path))

	cmd := exec.CommandContext(ctx, "ab", args...)
	return cmd
}

// parseOutput parses the benchmark tool output
func (r *Runner) parseOutput(output string, framework types.Framework, scenario types.Scenario, batch, retry int, duration time.Duration) (*types.TestResult, error) {
	if r.config.Benchmark.Tool == "wrk" {
		return r.parseWrkOutput(output, framework, scenario, batch, retry, duration)
	} else {
		return r.parseAbOutput(output, framework, scenario, batch, retry, duration)
	}
}

// parseWrkOutput parses wrk output
func (r *Runner) parseWrkOutput(output string, framework types.Framework, scenario types.Scenario, batch, retry int, duration time.Duration) (*types.TestResult, error) {
	// This is a simplified parser - in a real implementation, you'd want more robust parsing
	lines := strings.Split(output, "\n")

	result := &types.TestResult{
		Framework:   framework.Name,
		Scenario:    scenario.Name,
		Requests:    r.config.Benchmark.DefaultRequests,
		Connections: r.config.Benchmark.DefaultConnections,
		Duration:    duration,
		Timestamp:   time.Now(),
		Batch:       batch,
		Retry:       retry,
	}

	// Parse wrk output lines
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Requests/sec:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if rps, err := strconv.ParseFloat(parts[1], 64); err == nil {
					result.RequestsPerSec = rps
				}
			}
		}

		if strings.Contains(line, "Latency") && strings.Contains(line, "avg") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if latency, err := parseLatency(parts[1]); err == nil {
					result.LatencyMean = latency
				}
			}
		}
	}

	return result, nil
}

// parseAbOutput parses ApacheBench output
func (r *Runner) parseAbOutput(output string, framework types.Framework, scenario types.Scenario, batch, retry int, duration time.Duration) (*types.TestResult, error) {
	// This is a simplified parser - in a real implementation, you'd want more robust parsing
	lines := strings.Split(output, "\n")

	result := &types.TestResult{
		Framework:   framework.Name,
		Scenario:    scenario.Name,
		Requests:    r.config.Benchmark.DefaultRequests,
		Connections: r.config.Benchmark.DefaultConnections,
		Duration:    duration,
		Timestamp:   time.Now(),
		Batch:       batch,
		Retry:       retry,
	}

	// Parse ab output lines
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Requests per second:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if rps, err := strconv.ParseFloat(parts[3], 64); err == nil {
					result.RequestsPerSec = rps
				}
			}
		}

		if strings.Contains(line, "Time per request:") && strings.Contains(line, "mean") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if latency, err := parseLatency(parts[3]); err == nil {
					result.LatencyMean = latency
				}
			}
		}
	}

	return result, nil
}

// monitorTestProgress monitors the progress of a running test and shows periodic updates
func (r *Runner) monitorTestProgress(ctx context.Context, framework, scenario string, batch, retry int) {
	ticker := time.NewTicker(1 * time.Second) // Update every second for 1% granularity
	defer ticker.Stop()

	startTime := time.Now()

	// Estimate test duration based on configuration
	var estimatedDuration time.Duration
	if r.config.Benchmark.DefaultRequests > 0 {
		// For request-based tests, estimate based on expected RPS
		estimatedRPS := 50000.0 // Conservative estimate
		estimatedDuration = time.Duration(float64(r.config.Benchmark.DefaultRequests)/estimatedRPS) * time.Second
	} else {
		// For duration-based tests, parse the configured duration
		if duration, err := time.ParseDuration(r.config.Benchmark.DefaultDuration); err == nil {
			estimatedDuration = duration
		} else {
			estimatedDuration = 30 * time.Second // Default fallback
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed >= estimatedDuration {
				return // Test should be complete
			}

			percentage := float64(elapsed) / float64(estimatedDuration) * 100
			if percentage >= 1.0 { // Only show if at least 1%
				remaining := estimatedDuration - elapsed
				r.tracker.LogInfo("🔄 [%s] %s (Batch %d) - %.1f%% complete (ETA: %s)",
					framework, scenario, batch, percentage, remaining.Round(time.Second))
			}
		}
	}
}

// saveResults saves all results to CSV files
func (r *Runner) saveResults(resultsDir string) error {
	r.tracker.LogInfo("Saving results...")

	// Load all results from progress state to ensure we save everything
	state, err := r.tracker.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load progress state: %w", err)
	}

	// Use all results from progress state instead of just r.results
	allResults := state.Results
	if len(allResults) == 0 {
		r.tracker.LogWarning("No results found in progress state")
		allResults = r.results // Fallback to current session results
	}

	// Always create new CSV files (don't append) to ensure clean output
	r.tracker.LogInfo("Creating new results files with %d total results", len(allResults))

	// Save summary CSV
	summaryFile := filepath.Join(resultsDir, "summary.csv")
	if err := r.saveSummaryCSVWithResults(summaryFile, allResults); err != nil {
		return fmt.Errorf("failed to save summary CSV: %w", err)
	}

	// Save individual framework CSVs
	for frameworkName := range r.config.Frameworks {
		frameworkFile := filepath.Join(resultsDir, "parts", fmt.Sprintf("summary_%s.csv", frameworkName))
		if err := r.saveFrameworkCSVWithResults(frameworkFile, frameworkName, allResults); err != nil {
			return fmt.Errorf("failed to save framework CSV for %s: %w", frameworkName, err)
		}
	}

	r.tracker.LogSuccess("Results saved successfully")
	return nil
}

// saveSummaryCSVWithResults saves the summary CSV file with provided results
func (r *Runner) saveSummaryCSVWithResults(filename string, results []types.TestResult) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Framework", "Scenario", "Batch", "Retry", "Requests", "Connections",
		"Duration", "RequestsPerSec", "LatencyMean", "LatencyP50", "LatencyP90",
		"LatencyP99", "MaxLatency", "TransferRate", "Errors", "Non2XX", "Timestamp",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, result := range results {
		row := []string{
			result.Framework,
			result.Scenario,
			strconv.Itoa(result.Batch),
			strconv.Itoa(result.Retry),
			strconv.Itoa(result.Requests),
			strconv.Itoa(result.Connections),
			result.Duration.String(),
			fmt.Sprintf("%.2f", result.RequestsPerSec),
			result.LatencyMean.String(),
			result.LatencyP50.String(),
			result.LatencyP90.String(),
			result.LatencyP99.String(),
			result.MaxLatency.String(),
			fmt.Sprintf("%.2f", result.TransferRate),
			strconv.Itoa(result.Errors),
			strconv.Itoa(result.Non2XX),
			result.Timestamp.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// saveSummaryCSV saves the summary CSV file
func (r *Runner) saveSummaryCSV(filename string, append bool) error {
	var file *os.File
	var err error

	if append {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(filename)
	}

	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if not appending OR if file is empty
	writeHeader := !append
	if append {
		// Check if file is empty
		if stat, err := file.Stat(); err == nil && stat.Size() == 0 {
			writeHeader = true
		}
	}

	if writeHeader {
		header := []string{
			"Framework", "Scenario", "Batch", "Retry", "Requests", "Connections",
			"Duration", "RequestsPerSec", "LatencyMean", "LatencyP50", "LatencyP90",
			"LatencyP99", "MaxLatency", "TransferRate", "Errors", "Non2XX", "Timestamp",
		}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	// Write data
	for _, result := range r.results {
		row := []string{
			result.Framework,
			result.Scenario,
			strconv.Itoa(result.Batch),
			strconv.Itoa(result.Retry),
			strconv.Itoa(result.Requests),
			strconv.Itoa(result.Connections),
			result.Duration.String(),
			fmt.Sprintf("%.2f", result.RequestsPerSec),
			result.LatencyMean.String(),
			result.LatencyP50.String(),
			result.LatencyP90.String(),
			result.LatencyP99.String(),
			result.MaxLatency.String(),
			fmt.Sprintf("%.2f", result.TransferRate),
			strconv.Itoa(result.Errors),
			strconv.Itoa(result.Non2XX),
			result.Timestamp.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// saveFrameworkCSVWithResults saves a framework-specific CSV file with provided results
func (r *Runner) saveFrameworkCSVWithResults(filename, frameworkName string, results []types.TestResult) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Scenario", "Batch", "Retry", "Requests", "Connections",
		"Duration", "RequestsPerSec", "LatencyMean", "LatencyP50", "LatencyP90",
		"LatencyP99", "MaxLatency", "TransferRate", "Errors", "Non2XX", "Timestamp",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data for this framework
	for _, result := range results {
		if result.Framework == frameworkName {
			row := []string{
				result.Scenario,
				strconv.Itoa(result.Batch),
				strconv.Itoa(result.Retry),
				strconv.Itoa(result.Requests),
				strconv.Itoa(result.Connections),
				result.Duration.String(),
				fmt.Sprintf("%.2f", result.RequestsPerSec),
				result.LatencyMean.String(),
				result.LatencyP50.String(),
				result.LatencyP90.String(),
				result.LatencyP99.String(),
				result.MaxLatency.String(),
				fmt.Sprintf("%.2f", result.TransferRate),
				strconv.Itoa(result.Errors),
				strconv.Itoa(result.Non2XX),
				result.Timestamp.Format(time.RFC3339),
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

// saveFrameworkCSV saves a framework-specific CSV file
func (r *Runner) saveFrameworkCSV(filename, frameworkName string, append bool) error {
	var file *os.File
	var err error

	if append {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(filename)
	}

	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if not appending OR if file is empty
	writeHeader := !append
	if append {
		// Check if file is empty
		if stat, err := file.Stat(); err == nil && stat.Size() == 0 {
			writeHeader = true
		}
	}

	if writeHeader {
		header := []string{
			"Scenario", "Batch", "Retry", "Requests", "Connections",
			"Duration", "RequestsPerSec", "LatencyMean", "LatencyP50", "LatencyP90",
			"LatencyP99", "MaxLatency", "TransferRate", "Errors", "Non2XX", "Timestamp",
		}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	// Write data for this framework
	for _, result := range r.results {
		if result.Framework == frameworkName {
			row := []string{
				result.Scenario,
				strconv.Itoa(result.Batch),
				strconv.Itoa(result.Retry),
				strconv.Itoa(result.Requests),
				strconv.Itoa(result.Connections),
				result.Duration.String(),
				fmt.Sprintf("%.2f", result.RequestsPerSec),
				result.LatencyMean.String(),
				result.LatencyP50.String(),
				result.LatencyP90.String(),
				result.LatencyP99.String(),
				result.MaxLatency.String(),
				fmt.Sprintf("%.2f", result.TransferRate),
				strconv.Itoa(result.Errors),
				strconv.Itoa(result.Non2XX),
				result.Timestamp.Format(time.RFC3339),
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

// handleExistingResults handles existing CSV files based on progress state
func (r *Runner) handleExistingResults(resultsDir string, resumeInfo *types.ResumeInfo) error {
	// Check if there are existing CSV files
	summaryFile := filepath.Join(resultsDir, "summary.csv")
	partsDir := filepath.Join(resultsDir, "parts")

	// Clear existing results unless we're resuming an incomplete run
	shouldClear := true
	state, err := r.tracker.LoadState()
	if err == nil && state != nil && state.Status != "completed" {
		shouldClear = false
		r.tracker.LogInfo("Found incomplete progress state - will accumulate results in existing files")
	}

	if shouldClear {
		r.tracker.LogInfo("Starting fresh benchmark - clearing existing results")

		// Remove existing CSV files
		if err := os.Remove(summaryFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing summary.csv: %w", err)
		}

		// Remove existing parts directory
		if err := os.RemoveAll(partsDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing parts directory: %w", err)
		}

		// Recreate parts directory
		if err := os.MkdirAll(partsDir, 0755); err != nil {
			return fmt.Errorf("failed to recreate parts directory: %w", err)
		}

		r.tracker.LogInfo("Cleared existing results - ready for fresh benchmark")
	}

	return nil
}

// generateRunID generates a unique run ID
func generateRunID() string {
	return fmt.Sprintf("run_%s", time.Now().Format("20060102_150405"))
}

// parseLatency parses latency strings (e.g., "1.23ms", "456.78us")
func parseLatency(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Handle different units
	if strings.HasSuffix(s, "ms") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(s, "ms"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(val * float64(time.Millisecond)), nil
	}

	if strings.HasSuffix(s, "us") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(s, "us"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(val * float64(time.Microsecond)), nil
	}

	if strings.HasSuffix(s, "s") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(val * float64(time.Second)), nil
	}

	// Try parsing as milliseconds
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(val * float64(time.Millisecond)), nil
	}

	return 0, fmt.Errorf("unable to parse latency: %s", s)
}

// generateREADME generates README.md from template and creates visualizations
func (r *Runner) generateREADME(resultsDir string) error {
	r.tracker.LogInfo("Generating visualizations and README from template...")

	// First, generate charts using Python scripts
	if err := r.generateCharts(resultsDir); err != nil {
		r.tracker.LogWarning("Failed to generate charts: %v", err)
		// Continue with README generation even if charts fail
	}

	// Read the template file
	templatePath := "README.template.md"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Extract date from results directory path
	// resultsDir format: "results/2025-08-25"
	dateStr := filepath.Base(resultsDir)

	// Generate statistical tables from results
	overallRankingTable, err := r.generateOverallRankingTable()
	if err != nil {
		r.tracker.LogWarning("Failed to generate overall ranking table: %v", err)
		overallRankingTable = "*Statistics table generation failed*"
	}

	perScenarioTables, err := r.generatePerScenarioTables()
	if err != nil {
		r.tracker.LogWarning("Failed to generate per-scenario tables: %v", err)
		perScenarioTables = "*Per-scenario tables generation failed*"
	}

	totalTests := len(r.results)

	// Replace template placeholders
	content := string(templateContent)
	content = strings.ReplaceAll(content, "{{DATE}}", dateStr)
	content = strings.ReplaceAll(content, "{{TOTAL_TESTS}}", fmt.Sprintf("%d", totalTests))
	content = strings.ReplaceAll(content, "{{OVERALL_RANKING_TABLE}}", overallRankingTable)
	content = strings.ReplaceAll(content, "{{PER_SCENARIO_TABLES}}", perScenarioTables)

	// Update image paths to use 'images' instead of 'charts'
	content = strings.ReplaceAll(content, "/charts/", "/images/")

	// Write the rendered README to the results directory
	readmePath := filepath.Join(resultsDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write README file: %w", err)
	}

	r.tracker.LogSuccess("README.md generated successfully at %s", readmePath)
	return nil
}

// generateCharts runs Python scripts to generate visualizations
func (r *Runner) generateCharts(resultsDir string) error {
	r.tracker.LogInfo("Generating charts and visualizations...")

	// Check if .venv exists
	venvPath := ".venv"
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		return fmt.Errorf("virtual environment not found at %s. Please create it with: python3 -m venv .venv && source .venv/bin/activate && pip install matplotlib pandas numpy", venvPath)
	}

	// Prepare environment with venv activation
	venvPython := filepath.Join(venvPath, "bin", "python3")
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		return fmt.Errorf("python executable not found in venv: %s", venvPython)
	}

	// Run the chart generation script with venv python
	chartCmd := exec.Command(venvPython, "bin/load_and_render_csv.py", resultsDir)
	chartOutput, err := chartCmd.CombinedOutput()
	if err != nil {
		r.tracker.LogWarning("Chart generation failed: %v, output: %s", err, string(chartOutput))
		return fmt.Errorf("chart generation failed: %w", err)
	}
	r.tracker.LogInfo("Chart generation completed successfully")

	// Run the summary generation script with venv python
	summaryCmd := exec.Command(venvPython, "bin/build_final_summary.py", resultsDir)
	summaryOutput, err := summaryCmd.CombinedOutput()
	if err != nil {
		r.tracker.LogWarning("Summary generation failed: %v, output: %s", err, string(summaryOutput))
		return fmt.Errorf("summary generation failed: %w", err)
	}
	r.tracker.LogInfo("Summary generation completed successfully")

	r.tracker.LogSuccess("Charts and visualizations generated successfully")
	return nil
}

// generateOverallRankingTable creates a markdown table with overall framework rankings
func (r *Runner) generateOverallRankingTable() (string, error) {
	if len(r.results) == 0 {
		return "*No results available*", nil
	}

	// Calculate average RPS per framework across all scenarios
	frameworkStats := make(map[string][]float64)
	for _, result := range r.results {
		frameworkStats[result.Framework] = append(frameworkStats[result.Framework], result.RequestsPerSec)
	}

	// Calculate averages and create ranking
	type FrameworkRanking struct {
		Name      string
		AvgRPS    float64
		MinRPS    float64
		MaxRPS    float64
		TestCount int
		Rank      int
	}

	var rankings []FrameworkRanking
	for framework, rpsList := range frameworkStats {
		if len(rpsList) == 0 {
			continue
		}

		var sum, min, max float64
		min = rpsList[0]
		max = rpsList[0]

		for _, rps := range rpsList {
			sum += rps
			if rps < min {
				min = rps
			}
			if rps > max {
				max = rps
			}
		}

		rankings = append(rankings, FrameworkRanking{
			Name:      framework,
			AvgRPS:    sum / float64(len(rpsList)),
			MinRPS:    min,
			MaxRPS:    max,
			TestCount: len(rpsList),
		})
	}

	// Sort by average RPS (descending)
	for i := 0; i < len(rankings)-1; i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[i].AvgRPS < rankings[j].AvgRPS {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	// Assign ranks
	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	// Generate markdown table
	var table strings.Builder
	table.WriteString("| 🏆 Rank | Framework | Avg RPS | Min RPS | Max RPS | Tests | Performance |\n")
	table.WriteString("|---------|-----------|---------|---------|---------|-------|-------------|\n")

	for _, ranking := range rankings {
		var medal string
		switch ranking.Rank {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("#%d", ranking.Rank)
		}

		// Performance indicator
		var performance string
		if ranking.Rank == 1 {
			performance = "🔥 **Excellent**"
		} else if ranking.Rank <= 2 {
			performance = "⚡ **Very Good**"
		} else if ranking.Rank <= 3 {
			performance = "✅ **Good**"
		} else {
			performance = "📊 **Baseline**"
		}

		table.WriteString(fmt.Sprintf("| %s | **%s** | %s | %s | %s | %d | %s |\n",
			medal,
			ranking.Name,
			formatNumber(ranking.AvgRPS),
			formatNumber(ranking.MinRPS),
			formatNumber(ranking.MaxRPS),
			ranking.TestCount,
			performance,
		))
	}

	return table.String(), nil
}

// generatePerScenarioTables creates detailed tables for each scenario
func (r *Runner) generatePerScenarioTables() (string, error) {
	if len(r.results) == 0 {
		return "*No results available*", nil
	}

	// Group results by scenario
	scenarioResults := make(map[string][]types.TestResult)
	for _, result := range r.results {
		scenarioResults[result.Scenario] = append(scenarioResults[result.Scenario], result)
	}

	var allTables strings.Builder

	for scenario, results := range scenarioResults {
		// Create framework performance map for this scenario
		frameworkPerf := make(map[string][]float64)
		for _, result := range results {
			frameworkPerf[result.Framework] = append(frameworkPerf[result.Framework], result.RequestsPerSec)
		}

		// Calculate averages and sort
		type ScenarioRanking struct {
			Framework string
			AvgRPS    float64
			Rank      int
		}

		var rankings []ScenarioRanking
		for framework, rpsList := range frameworkPerf {
			if len(rpsList) == 0 {
				continue
			}
			var sum float64
			for _, rps := range rpsList {
				sum += rps
			}
			rankings = append(rankings, ScenarioRanking{
				Framework: framework,
				AvgRPS:    sum / float64(len(rpsList)),
			})
		}

		// Sort by RPS (descending)
		for i := 0; i < len(rankings)-1; i++ {
			for j := i + 1; j < len(rankings); j++ {
				if rankings[i].AvgRPS < rankings[j].AvgRPS {
					rankings[i], rankings[j] = rankings[j], rankings[i]
				}
			}
		}

		// Assign ranks
		for i := range rankings {
			rankings[i].Rank = i + 1
		}

		// Generate table for this scenario
		allTables.WriteString(fmt.Sprintf("\n#### 📊 %s Performance\n\n", scenario))
		allTables.WriteString("| 🏆 Rank | Framework | Avg RPS | Performance vs Leader |\n")
		allTables.WriteString("|---------|-----------|---------|----------------------|\n")

		leaderRPS := float64(0)
		if len(rankings) > 0 {
			leaderRPS = rankings[0].AvgRPS
		}

		for _, ranking := range rankings {
			var medal string
			switch ranking.Rank {
			case 1:
				medal = "🥇"
			case 2:
				medal = "🥈"
			case 3:
				medal = "🥉"
			default:
				medal = fmt.Sprintf("#%d", ranking.Rank)
			}

			var vsLeader string
			if ranking.Rank == 1 {
				vsLeader = "**100%** (Leader)"
			} else if leaderRPS > 0 {
				percentage := (ranking.AvgRPS / leaderRPS) * 100
				vsLeader = fmt.Sprintf("%.1f%% of leader", percentage)
			} else {
				vsLeader = "N/A"
			}

			allTables.WriteString(fmt.Sprintf("| %s | **%s** | %s | %s |\n",
				medal,
				ranking.Framework,
				formatNumber(ranking.AvgRPS),
				vsLeader,
			))
		}
		allTables.WriteString("\n")
	}

	return allTables.String(), nil
}

// formatNumber formats a number with thousand separators
func formatNumber(n float64) string {
	// Format number and add thousand separators manually
	str := fmt.Sprintf("%.0f", n)

	// Add thousand separators
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	return result.String()
}
