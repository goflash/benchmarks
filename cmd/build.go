package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goflash/benchmarks/internal/config"
	"github.com/goflash/benchmarks/internal/progress"
	"github.com/goflash/benchmarks/internal/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build all framework servers",
	Long:  `Build all framework servers for benchmarking. This command compiles optimized production builds of all configured frameworks.`,
	RunE:  runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolP("clean", "c", false, "Clean build directory before building")
	buildCmd.Flags().StringP("output", "o", "build", "Output directory for binaries")
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get flags
	clean, _ := cmd.Flags().GetBool("clean")
	outputDir, _ := cmd.Flags().GetString("output")

	// Create progress tracker
	tracker := progress.NewTracker(viper.GetBool("verbose"))
	tracker.PrintHeader()

	if clean {
		tracker.LogInfo("Cleaning build directory...")
		if err := os.RemoveAll(outputDir); err != nil {
			return fmt.Errorf("failed to clean build directory: %w", err)
		}
	}

	// Create build directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	tracker.LogInfo("Building framework servers...")

	// Build each framework
	for name, framework := range cfg.Frameworks {
		tracker.LogInfo("Building %s...", framework.Name)

		if err := buildFramework(framework, outputDir); err != nil {
			return fmt.Errorf("failed to build %s: %w", name, err)
		}

		tracker.LogSuccess("Built %s successfully", framework.Name)
	}

	tracker.LogSuccess("All frameworks built successfully")
	return nil
}

func buildFramework(framework types.Framework, outputDir string) error {
	// Change to framework directory
	if err := os.Chdir(framework.BuildPath); err != nil {
		return fmt.Errorf("failed to change to framework directory: %w", err)
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", filepath.Join("..", "..", outputDir, framework.BinaryName), ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build binary: %w", err)
	}

	// Change back to original directory
	if err := os.Chdir("../.."); err != nil {
		return fmt.Errorf("failed to change back to original directory: %w", err)
	}

	return nil
}
