package types

import (
	"time"
)

// Framework represents a web framework configuration
type Framework struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Port        int    `mapstructure:"port"`
	URL         string `mapstructure:"url"`
	BuildPath   string `mapstructure:"build_path"`
	BinaryName  string `mapstructure:"binary_name"`
	Description string `mapstructure:"description"`
}

// Scenario represents a test scenario configuration
type Scenario struct {
	Name           string `mapstructure:"name"`
	Method         string `mapstructure:"method"`
	Path           string `mapstructure:"path"`
	Description    string `mapstructure:"description"`
	ExpectedStatus int    `mapstructure:"expected_status"`
	BodyFile       string `mapstructure:"body_file,omitempty"`
}

// BenchmarkConfig represents benchmark tool configuration
type BenchmarkConfig struct {
	Tool                string  `mapstructure:"tool"`
	Threads             int     `mapstructure:"threads"`
	Timeout             string  `mapstructure:"timeout"`
	KeepAlive           bool    `mapstructure:"keep_alive"`
	DefaultRequests     int     `mapstructure:"default_requests"`
	DefaultConnections  int     `mapstructure:"default_connections"`
	DefaultDuration     string  `mapstructure:"default_duration"`
	Batches             int     `mapstructure:"batches"`
	MaxRetries          int     `mapstructure:"max_retries"`
	RetrySleep          int     `mapstructure:"retry_sleep"`
	BatchPause          int     `mapstructure:"batch_pause"`
	AllowSocketErrors   int     `mapstructure:"allow_socket_errors"`
	AllowNon2XX         int     `mapstructure:"allow_non_2xx"`
	HealthCheckTimeout  int     `mapstructure:"health_check_timeout"`
	HealthCheckInterval float64 `mapstructure:"health_check_interval"`
}

// OutputConfig represents output configuration
type OutputConfig struct {
	ResultsDir        string `mapstructure:"results_dir"`
	LogsDir           string `mapstructure:"logs_dir"`
	DateFormat        string `mapstructure:"date_format"`
	CSVIncludeHeaders bool   `mapstructure:"csv_include_headers"`
	CSVDelimiter      string `mapstructure:"csv_delimiter"`
	ShowProgress      bool   `mapstructure:"show_progress"`
	ProgressInterval  int    `mapstructure:"progress_interval"`
}

// SystemConfig represents system configuration
type SystemConfig struct {
	MaxFileDescriptors int  `mapstructure:"max_file_descriptors"`
	ProcessTimeout     int  `mapstructure:"process_timeout"`
	CleanupOnExit      bool `mapstructure:"cleanup_on_exit"`
	MonitorResources   bool `mapstructure:"monitor_resources"`
	ResourceInterval   int  `mapstructure:"resource_interval"`
}

// ProcessConfig represents process management configuration
type ProcessConfig struct {
	MaxRestarts         int     `mapstructure:"max_restarts"`
	RestartDelay        int     `mapstructure:"restart_delay_seconds"`
	BackoffMultiplier   float64 `mapstructure:"backoff_multiplier"`
	MaxRestartDelay     int     `mapstructure:"max_restart_delay_seconds"`
	HealthCheckInterval int     `mapstructure:"health_check_interval_seconds"`
	StartupTimeout      int     `mapstructure:"startup_timeout_seconds"`
	ShutdownTimeout     int     `mapstructure:"shutdown_timeout_seconds"`
}

// Config represents the complete configuration structure
type Config struct {
	Frameworks map[string]Framework `mapstructure:"frameworks"`
	Scenarios  map[string]Scenario  `mapstructure:"scenarios"`
	Benchmark  BenchmarkConfig      `mapstructure:"benchmark"`
	Output     OutputConfig         `mapstructure:"output"`
	System     SystemConfig         `mapstructure:"system"`
	Process    ProcessConfig        `mapstructure:"process"`
}

// TestResult represents a single test result
type TestResult struct {
	Framework      string
	Scenario       string
	Requests       int
	Connections    int
	Duration       time.Duration
	RequestsPerSec float64
	LatencyMean    time.Duration
	LatencyP50     time.Duration
	LatencyP90     time.Duration
	LatencyP99     time.Duration
	MaxLatency     time.Duration
	TransferRate   float64
	Errors         int
	Non2XX         int
	Timestamp      time.Time
	Batch          int
	Retry          int
}

// TestRun represents a complete test run
type TestRun struct {
	ID           string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Results      []TestResult
	Config       Config
	Status       string
	ErrorMessage string
}

// ProgressInfo represents progress information
type ProgressInfo struct {
	CurrentFramework string
	CurrentScenario  string
	CurrentBatch     int
	TotalBatches     int
	CurrentRetry     int
	MaxRetries       int
	CompletedTests   int
	TotalTests       int
	StartTime        time.Time
	EstimatedEnd     time.Time
}

// ResumeInfo represents resume information for failed runs
type ResumeInfo struct {
	RunID          string
	LastFramework  string
	LastScenario   string
	LastBatch      int
	LastRetry      int
	CompletedTests []string
	FailedTests    []string
	ResultsDir     string
	Config         Config
}
