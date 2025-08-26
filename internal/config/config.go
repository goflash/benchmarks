package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goflash/benchmarks/internal/types"
	"github.com/spf13/viper"
)

// Loader handles configuration loading and validation
type Loader struct {
	config *types.Config
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load loads the configuration from the specified file or default location
func (l *Loader) Load(configFile string) (*types.Config, error) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config types.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults and validate
	if err := l.setDefaults(&config); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	if err := l.validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	l.config = &config
	return &config, nil
}

// setDefaults sets default values for missing configuration
func (l *Loader) setDefaults(config *types.Config) error {
	// Set default output directory if not specified
	if config.Output.ResultsDir == "" {
		config.Output.ResultsDir = "results"
	}

	if config.Output.LogsDir == "" {
		config.Output.LogsDir = "logs"
	}

	if config.Output.DateFormat == "" {
		config.Output.DateFormat = "2006-01-02"
	}

	// Set default benchmark tool if not specified
	if config.Benchmark.Tool == "" {
		config.Benchmark.Tool = "wrk"
	}

	// Set default threads if not specified
	if config.Benchmark.Threads == 0 {
		config.Benchmark.Threads = 11
	}

	// Set default timeout if not specified
	if config.Benchmark.Timeout == "" {
		config.Benchmark.Timeout = "10s"
	}

	// Set default requests if not specified
	// Note: 0 means use duration-based testing instead
	if config.Benchmark.DefaultRequests == 0 {
		config.Benchmark.DefaultRequests = 0 // Use duration-based testing by default
	}

	// Set default connections if not specified
	if config.Benchmark.DefaultConnections == 0 {
		config.Benchmark.DefaultConnections = 256
	}

	// Set default duration if not specified
	if config.Benchmark.DefaultDuration == "" {
		config.Benchmark.DefaultDuration = "30s"
	}

	// Set default batches if not specified
	if config.Benchmark.Batches == 0 {
		config.Benchmark.Batches = 3
	}

	// Set default max retries if not specified
	if config.Benchmark.MaxRetries == 0 {
		config.Benchmark.MaxRetries = 3
	}

	// Set default retry sleep if not specified
	if config.Benchmark.RetrySleep == 0 {
		config.Benchmark.RetrySleep = 2
	}

	// Set default batch pause if not specified
	if config.Benchmark.BatchPause == 0 {
		config.Benchmark.BatchPause = 5
	}

	// Set default health check timeout if not specified
	if config.Benchmark.HealthCheckTimeout == 0 {
		config.Benchmark.HealthCheckTimeout = 30
	}

	// Set default health check interval if not specified
	if config.Benchmark.HealthCheckInterval == 0 {
		config.Benchmark.HealthCheckInterval = 0.1
	}

	// Set default max file descriptors if not specified
	if config.System.MaxFileDescriptors == 0 {
		config.System.MaxFileDescriptors = 65536
	}

	// Set default process timeout if not specified
	if config.System.ProcessTimeout == 0 {
		config.System.ProcessTimeout = 300
	}

	// Set default resource interval if not specified
	if config.System.ResourceInterval == 0 {
		config.System.ResourceInterval = 5
	}

	return nil
}

// validate validates the configuration
func (l *Loader) validate(config *types.Config) error {
	// Validate frameworks
	if len(config.Frameworks) == 0 {
		return fmt.Errorf("no frameworks configured")
	}

	for name, framework := range config.Frameworks {
		if framework.Name == "" {
			return fmt.Errorf("framework %s: name is required", name)
		}
		if framework.Port == 0 {
			return fmt.Errorf("framework %s: port is required", name)
		}
		if framework.URL == "" {
			return fmt.Errorf("framework %s: URL is required", name)
		}
		if framework.BuildPath == "" {
			return fmt.Errorf("framework %s: build path is required", name)
		}
		if framework.BinaryName == "" {
			return fmt.Errorf("framework %s: binary name is required", name)
		}
	}

	// Validate scenarios
	if len(config.Scenarios) == 0 {
		return fmt.Errorf("no scenarios configured")
	}

	for name, scenario := range config.Scenarios {
		if scenario.Name == "" {
			return fmt.Errorf("scenario %s: name is required", name)
		}
		if scenario.Method == "" {
			return fmt.Errorf("scenario %s: method is required", name)
		}
		if scenario.Path == "" {
			return fmt.Errorf("scenario %s: path is required", name)
		}
		if scenario.ExpectedStatus == 0 {
			return fmt.Errorf("scenario %s: expected status is required", name)
		}
	}

	// Validate benchmark tool
	if config.Benchmark.Tool != "wrk" && config.Benchmark.Tool != "ab" {
		return fmt.Errorf("unsupported benchmark tool: %s (supported: wrk, ab)", config.Benchmark.Tool)
	}

	// Validate benchmark parameters
	// Note: DefaultRequests can be 0 (use duration-based testing) or positive (use request-based testing)
	if config.Benchmark.DefaultRequests < 0 {
		return fmt.Errorf("default requests cannot be negative")
	}
	if config.Benchmark.DefaultConnections <= 0 {
		return fmt.Errorf("default connections must be positive")
	}
	if config.Benchmark.Batches <= 0 {
		return fmt.Errorf("batches must be positive")
	}
	if config.Benchmark.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	// Validate output configuration
	if config.Output.ResultsDir == "" {
		return fmt.Errorf("results directory is required")
	}

	// Validate system configuration
	if config.System.MaxFileDescriptors <= 0 {
		return fmt.Errorf("max file descriptors must be positive")
	}
	if config.System.ProcessTimeout <= 0 {
		return fmt.Errorf("process timeout must be positive")
	}

	return nil
}

// GetConfig returns the loaded configuration
func (l *Loader) GetConfig() *types.Config {
	return l.config
}

// CreateResultsDir creates the results directory with date-based subdirectory
func (l *Loader) CreateResultsDir() (string, error) {
	if l.config == nil {
		return "", fmt.Errorf("config not loaded")
	}

	dateStr := time.Now().Format(l.config.Output.DateFormat)
	resultsDir := filepath.Join(l.config.Output.ResultsDir, dateStr)

	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create results directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"raw", "parts", "logs", "charts"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(resultsDir, subdir)
		if err := os.MkdirAll(subdirPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create subdirectory %s: %w", subdir, err)
		}
	}

	return resultsDir, nil
}

// GetFramework returns a framework by name
func (l *Loader) GetFramework(name string) (*types.Framework, error) {
	if l.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	framework, exists := l.config.Frameworks[name]
	if !exists {
		return nil, fmt.Errorf("framework %s not found", name)
	}

	return &framework, nil
}

// GetScenario returns a scenario by name
func (l *Loader) GetScenario(name string) (*types.Scenario, error) {
	if l.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	scenario, exists := l.config.Scenarios[name]
	if !exists {
		return nil, fmt.Errorf("scenario %s not found", name)
	}

	return &scenario, nil
}

// GetFrameworks returns all configured frameworks
func (l *Loader) GetFrameworks() map[string]types.Framework {
	if l.config == nil {
		return nil
	}
	return l.config.Frameworks
}

// GetScenarios returns all configured scenarios
func (l *Loader) GetScenarios() map[string]types.Scenario {
	if l.config == nil {
		return nil
	}
	return l.config.Scenarios
}
