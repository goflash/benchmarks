package main

import (
	"context"
	"fmt"
	"time"

	"github.com/goflash/benchmarks/internal/config"
	"github.com/goflash/benchmarks/internal/progress"
	"github.com/goflash/benchmarks/internal/runner"
	"github.com/goflash/benchmarks/internal/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run benchmark tests",
	Long:  `Run the complete benchmark suite against all configured frameworks and scenarios.`,
	RunE:  runBenchmarks,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntP("requests", "n", 0, "Number of requests per test (overrides config)")
	runCmd.Flags().IntP("connections", "c", 0, "Number of concurrent connections (overrides config)")
	runCmd.Flags().String("duration", "", "Test duration (overrides config)")
	runCmd.Flags().StringP("tool", "t", "", "Benchmark tool (wrk or ab, overrides config)")
	runCmd.Flags().IntP("batches", "b", 0, "Number of batches (overrides config)")
	runCmd.Flags().Int("retries", 0, "Maximum retries (overrides config)")
	runCmd.Flags().StringSliceP("frameworks", "f", nil, "Specific frameworks to test (overrides config)")
	runCmd.Flags().StringSliceP("scenarios", "s", nil, "Specific scenarios to test (overrides config)")
	runCmd.Flags().BoolP("resume", "", false, "Resume from last failed run")
}

func runBenchmarks(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override configuration with command line flags
	if err := overrideConfig(cmd, cfg); err != nil {
		return fmt.Errorf("failed to override config: %w", err)
	}

	// Create progress tracker
	tracker := progress.NewTracker(viper.GetBool("verbose"))

	// Create runner
	benchmarkRunner := runner.NewRunner(cfg, tracker)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.System.ProcessTimeout)*time.Second)
	defer cancel()

	// Run benchmarks
	run, err := benchmarkRunner.Run(ctx)
	if err != nil {
		return fmt.Errorf("benchmark run failed: %w", err)
	}

	if run.Status == "completed" {
		fmt.Printf("\nBenchmark completed successfully in %s\n", run.Duration)
		fmt.Printf("Results saved to: %s\n", run.ID)
	} else {
		return fmt.Errorf("benchmark failed: %s", run.ErrorMessage)
	}

	return nil
}

func overrideConfig(cmd *cobra.Command, cfg *types.Config) error {
	// Override requests (only if flag was explicitly set)
	if cmd.Flags().Changed("requests") {
		if requests, _ := cmd.Flags().GetInt("requests"); requests > 0 {
			cfg.Benchmark.DefaultRequests = requests
		}
	}

	// Override connections (only if flag was explicitly set)
	if cmd.Flags().Changed("connections") {
		if connections, _ := cmd.Flags().GetInt("connections"); connections > 0 {
			cfg.Benchmark.DefaultConnections = connections
		}
	}

	// Override duration
	if duration, _ := cmd.Flags().GetString("duration"); duration != "" {
		cfg.Benchmark.DefaultDuration = duration
	}

	// Override tool
	if tool, _ := cmd.Flags().GetString("tool"); tool != "" {
		if tool != "wrk" && tool != "ab" {
			return fmt.Errorf("unsupported benchmark tool: %s (supported: wrk, ab)", tool)
		}
		cfg.Benchmark.Tool = tool
	}

	// Override batches (only if flag was explicitly set)
	if cmd.Flags().Changed("batches") {
		if batches, _ := cmd.Flags().GetInt("batches"); batches > 0 {
			cfg.Benchmark.Batches = batches
		}
	}

	// Override retries (only if flag was explicitly set)
	if cmd.Flags().Changed("retries") {
		if retries, _ := cmd.Flags().GetInt("retries"); retries >= 0 {
			cfg.Benchmark.MaxRetries = retries
		}
	}

	// Override frameworks
	if frameworks, _ := cmd.Flags().GetStringSlice("frameworks"); len(frameworks) > 0 {
		filteredFrameworks := make(map[string]types.Framework)
		for _, name := range frameworks {
			if framework, exists := cfg.Frameworks[name]; exists {
				filteredFrameworks[name] = framework
			} else {
				return fmt.Errorf("framework not found: %s", name)
			}
		}
		cfg.Frameworks = filteredFrameworks
	}

	// Override scenarios
	if scenarios, _ := cmd.Flags().GetStringSlice("scenarios"); len(scenarios) > 0 {
		filteredScenarios := make(map[string]types.Scenario)
		for _, name := range scenarios {
			if scenario, exists := cfg.Scenarios[name]; exists {
				filteredScenarios[name] = scenario
			} else {
				return fmt.Errorf("scenario not found: %s", name)
			}
		}
		cfg.Scenarios = filteredScenarios
	}

	return nil
}
