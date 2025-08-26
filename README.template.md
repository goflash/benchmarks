# ⚡ Go Web Framework Benchmarks

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/Benchmark-Performance-FF6B6B?style=for-the-badge" alt="Benchmark" />
  <img src="https://img.shields.io/badge/Platform-macOS-000000?style=for-the-badge&logo=apple&logoColor=white" alt="Platform" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License" />
</div>

<p align="center">
  <strong>🚀 Comprehensive performance benchmarking suite comparing Go web frameworks with atomic, deterministic, and resumable test execution.</strong>
</p>

---

## 📋 Table of Contents

- [🎯 Overview](#-overview)
- [🏗️ Framework Comparison](#️-framework-comparison)
- [📊 Benchmark Scenarios](#-benchmark-scenarios)
- [🧪 Test Environment](#-test-environment)
- [📈 Results](#-results)
- [🚀 Quick Start](#-quick-start)
- [⚙️ Configuration](#️-configuration)
- [📚 Documentation](#-documentation)
- [🤝 Contributing](#-contributing)

## 🎯 Overview

This repository contains a comprehensive benchmarking suite designed to evaluate the performance of **Go web frameworks** with a focus on atomic, deterministic, and resumable test execution. Our goal is to provide accurate, reproducible, and meaningful performance comparisons across various real-world scenarios.

### 🏆 Frameworks Under Test

| Framework     | Version | Description                                               |
| ------------- | ------- | --------------------------------------------------------- |
| **🔥 GoFlash** | Latest  | High-performance, minimalist Go web framework             |
| **🍸 Gin**     | Latest  | Fast HTTP web framework with martini-like API             |
| **🕷️ Fiber**   | v2.52.0 | Express-inspired web framework built on Fasthttp          |
| **📢 Echo**    | v4.11.4 | High performance, extensible, minimalist Go web framework |
| **🔗 Chi**     | v5.0.11 | Lightweight, expressive and scalable HTTP router          |

## 🏗️ Framework Comparison

### ⚡ Performance Characteristics

- **GoFlash**: Optimized for speed with minimal overhead
- **Gin**: Battle-tested with excellent middleware ecosystem
- **Fiber**: Express.js-like API with high performance
- **Echo**: High performance with extensible middleware
- **Chi**: Lightweight and expressive routing

### 🎯 Use Case Alignment

Each framework excels in different scenarios, making this benchmark crucial for informed decision-making in your next Go project.

## 📊 Benchmark Scenarios

Our benchmark suite covers **9 comprehensive scenarios** that represent common web application patterns:

<details>
<summary><strong>📝 Click to expand scenario details</strong></summary>

| #   | Scenario               | Description                          | Real-world Impact              |
| --- | ---------------------- | ------------------------------------ | ------------------------------ |
| 1️⃣   | **Simple Ping/Pong**   | Basic endpoint response              | Foundation performance         |
| 2️⃣   | **URL Path Parameter** | Dynamic route parsing                | RESTful API endpoints          |
| 3️⃣   | **Request Context**    | Context read/write operations        | State management               |
| 4️⃣   | **JSON Binding**       | Request deserialization + validation | API data processing            |
| 5️⃣   | **Wildcard Routing**   | Trailing wildcard route matching     | File serving, catch-all routes |
| 6️⃣   | **Route Groups**       | Basic route organization             | API versioning                 |
| 7️⃣   | **Deep Route Groups**  | 10-level nested groups               | Complex routing hierarchies    |
| 8️⃣   | **Single Middleware**  | Basic middleware processing          | Authentication, logging        |
| 9️⃣   | **Middleware Chain**   | 10-middleware processing chain       | Complex request pipelines      |

</details>

## 🧪 Test Environment

### 🖥️ Hardware Specifications

- **Machine**: Apple MacBook Pro (M3 chip)
- **Memory**: 32 GB RAM
- **Architecture**: ARM64

### 🔧 Benchmarking Tools

- **Load Generator**: [wrk](https://github.com/wg/wrk) HTTP benchmarking tool
- **Threads**: 4 concurrent threads
- **Connections**: 50 concurrent connections
- **Protocol**: HTTP/1.1 with keep-alive

### 📐 Methodology

- ✅ Functionally equivalent handlers across all frameworks
- ✅ Production/release build settings enabled
- ✅ Consistent routing patterns and middleware implementation
- ✅ Multiple test runs for statistical significance
- ✅ Isolated server processes to prevent interference
- ✅ Atomic and deterministic test execution
- ✅ Resume capability from failed runs

> ⚠️ **Note**: Results are indicative and may vary based on workload, configuration, and environment. Always benchmark in your specific use case.

## 📈 Results

> 📊 **Complete dataset available**: Detailed CSV files and additional metrics can be found in the [`results/{{DATE}}/`](./results/{{DATE}}/) directory.

### 🏆 Overall Performance Rankings

Our comprehensive benchmarks reveal significant performance differences across frameworks and scenarios. Below are the key findings from **{{TOTAL_TESTS}}** total benchmark tests:

{{OVERALL_RANKING_TABLE}}

### 📊 Cumulative Performance Comparison

<div align="center">

![Cumulative Benchmark Results](./results/{{DATE}}/images/all_benchmarks.png)

*Higher bars indicate better performance (requests per second)*

</div>

### 📋 Per-Scenario Performance Analysis

{{PER_SCENARIO_TABLES}}

---

### 📋 Detailed Scenario Results

<details>
<summary><strong>🎯 Simple Ping/Pong Endpoint</strong></summary>

**Test**: Basic HTTP GET response without any processing

![Simple Ping/Pong Results](./results/{{DATE}}/images/simple_ping_pong_rps.png)

**Key Insights**:

- Foundation performance comparison
- Measures framework overhead
- Critical for high-throughput applications

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_simple.csv)

</details>

<details>
<summary><strong>🔗 URL Path Parameter Extraction</strong></summary>

**Test**: Dynamic route matching and parameter extraction (`/user/:id`)

![URL Parameter Results](./results/{{DATE}}/images/url_path_parameter_rps.png)

**Key Insights**:

- RESTful API performance
- Router efficiency comparison
- Path parsing overhead analysis

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_param.csv)

</details>

<details>
<summary><strong>📝 Request Context Operations</strong></summary>

**Test**: Writing to and reading from request context

![Context Operations Results](./results/{{DATE}}/images/request_context_rps.png)

**Key Insights**:

- Context management efficiency
- State preservation performance
- Middleware communication overhead

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_context.csv)

</details>

<details>
<summary><strong>📦 JSON Binding & Validation</strong></summary>

**Test**: JSON request deserialization with struct binding and validation

![JSON Binding Results](./results/{{DATE}}/images/json_binding_rps.png)

**Key Insights**:

- API data processing performance
- Serialization/deserialization efficiency
- Validation overhead impact

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_json.csv)

</details>

<details>
<summary><strong>🌟 Wildcard Route Parsing</strong></summary>

**Test**: Trailing wildcard route matching (`/files/*path`)

![Wildcard Routing Results](./results/{{DATE}}/images/wildcard_routing_rps.png)

**Key Insights**:

- File serving performance
- Catch-all route efficiency
- Dynamic path handling

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_wildcard.csv)

</details>

<details>
<summary><strong>📁 Route Groups</strong></summary>

**Test**: Basic route group organization (`/api/v1/users`)

![Route Groups Results](./results/{{DATE}}/images/route_groups_rps.png)

**Key Insights**:

- API organization efficiency
- Group routing overhead
- Nested structure performance

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_groups.csv)

</details>

<details>
<summary><strong>🏗️ Deep Route Groups (10 Levels)</strong></summary>

**Test**: Complex nested route groups (`/g1/g2/.../g10/endpoint`)

![Deep Route Groups Results](./results/{{DATE}}/images/deep_route_groups_rps.png)

**Key Insights**:

- Complex routing hierarchy performance
- Deep nesting overhead
- Scalability under complex structures

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_deepgroups.csv)

</details>

<details>
<summary><strong>⚙️ Single Middleware</strong></summary>

**Test**: Basic middleware processing (e.g., request logging)

![Single Middleware Results](./results/{{DATE}}/images/single_middleware_rps.png)

**Key Insights**:

- Middleware overhead analysis
- Basic processing pipeline performance
- Authentication/logging impact

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_middleware.csv)

</details>

<details>
<summary><strong>🔗 Middleware Chain (10 Middlewares)</strong></summary>

**Test**: Complex middleware chain with 10 sequential middlewares

![Middleware Chain Results](./results/{{DATE}}/images/middleware_chain_rps.png)

**Key Insights**:

- Complex pipeline performance
- Cumulative middleware overhead
- Enterprise-grade processing chains

**Results**: [CSV Data](./results/{{DATE}}/parts/summary_mw10.csv)

</details>

---

### 🌐 Server Configuration

| Framework     | Port     | Optimization        |
| ------------- | -------- | ------------------- |
| 🔥 **GoFlash** | `:17780` | Production mode     |
| 🍸 **Gin**     | `:17781` | Release mode        |
| 🕷️ **Fiber**   | `:17782` | Production settings |
| 📢 **Echo**    | `:17783` | Production mode     |
| 🔗 **Chi**     | `:17784` | Release mode        |

## 🚀 Quick Start

Get up and running with the benchmark suite in minutes! Follow these step-by-step instructions:

### 📋 Prerequisites

- **Go 1.21+** installed and configured
- **wrk** HTTP benchmarking tool
- **macOS/Linux** environment (recommended)

<details>
<summary><strong>🛠️ Installing Prerequisites</strong></summary>

#### Install wrk (macOS)

```bash
brew install wrk
```

#### Install wrk (Ubuntu/Debian)

```bash
sudo apt-get install wrk
```

</details>

### 🏁 Quick Setup

#### 1️⃣ Build All Framework Servers

```bash
# Build all framework servers
./benchmark build
```

This command will:

- 📦 Download dependencies for all frameworks
- 🔨 Compile optimized production builds
- 📁 Place executables in `build/` directory

#### 2️⃣ Run Performance Benchmarks

```bash
# 🏆 High-Volume Load Testing (1M requests, 10 batches for statistical significance)
go run ./cmd run --requests 1000000 --connections 100 --batches 10

# ⏱️ Duration-Based Testing (1 minute per test scenario)
go run ./cmd run --duration 1m --connections 50 --batches 3

# 🚀 Full benchmark suite (recommended for comprehensive analysis)
go run ./cmd run --requests 10000 --connections 50 --batches 3

# ⚡ Quick test (faster execution for development)
go run ./cmd run --requests 1000 --connections 10 --batches 1

# 🎯 Custom framework and scenario selection
go run ./cmd run --duration 30s --frameworks flash,gin,gofiber --scenarios simple,json,param

# 📊 Specific test configuration examples
go run ./cmd run --requests <requests> --connections <connections> --batches <batches>
go run ./cmd run --duration <duration> --connections <connections> --batches <batches>
```

**Parameters:**

- `--requests`: Total number of requests per scenario (use `0` for duration-based testing)
- `--duration`: Test duration per scenario (e.g., `30s`, `1m`, `5m`)
- `--connections`: Concurrent connections
- `--batches`: Number of test batches for statistical significance
- `--frameworks`: Comma-separated list of frameworks to test (e.g., `flash,gin,gofiber`)
- `--scenarios`: Comma-separated list of scenarios to run (e.g., `simple,json,param`)

#### 3️⃣ View Results

After running benchmarks, you'll find detailed results in the `results/` directory:

```
results/
├── 📊 {{DATE}}/                    # Date-based results directory
│   ├── 📈 summary.csv              # Comprehensive comparison data
│   ├── 📋 parts/                   # Individual framework results
│   ├── 🔍 raw/                     # Raw benchmark outputs
│   └── 📁 images/                  # Generated charts
└── 📁 previous-runs/               # Historical results
```

### ⚡ Performance Tips

<details>
<summary><strong>🔧 Optimization Recommendations</strong></summary>

#### For More Accurate Results

1. **Close unnecessary applications** to reduce system noise
2. **Run multiple batches** for statistical significance
3. **Use consistent system load** across test runs
4. **Monitor system resources** during benchmarks

#### Scaling Parameters

- **Light testing**: `--requests 1000 --connections 10`
- **Standard testing**: `--requests 10000 --connections 50`
- **Heavy testing**: `--requests 100000 --connections 100`

#### System Tuning

```bash
# Increase file descriptor limit (if needed)
ulimit -n 65536

# Check current limits
ulimit -a
```

</details>

---

## ⚙️ Configuration

### 🌐 Server Ports & Endpoints

| Framework     | Port    | Health Check | Base URL                 |
| ------------- | ------- | ------------ | ------------------------ |
| 🔥 **GoFlash** | `17780` | `GET /ping`  | `http://localhost:17780` |
| 🍸 **Gin**     | `17781` | `GET /ping`  | `http://localhost:17781` |
| 🕷️ **Fiber**   | `17782` | `GET /ping`  | `http://localhost:17782` |
| 📢 **Echo**    | `17783` | `GET /ping`  | `http://localhost:17783` |
| 🔗 **Chi**     | `17784` | `GET /ping`  | `http://localhost:17784` |

### 📝 Available Endpoints

Each server implements the following endpoints for benchmarking:

```
GET  /ping                    # Simple ping/pong
GET  /param/:id               # URL parameter extraction  
GET  /context                 # Request context operations
POST /json                    # JSON binding & validation
GET  /wildcard/*path          # Wildcard route parsing
GET  /api/v1/group/ping       # Basic route group
GET  /g1/g2/.../g10/ping      # Deep nested groups (10 levels)
GET  /mw/ping                 # Single middleware
GET  /mw10/ping               # 10 middleware chain
```

### 🔧 Benchmark Parameters

Customize benchmark execution with these parameters:

| Parameter       | Description             | Default | Recommended Range |
| --------------- | ----------------------- | ------- | ----------------- |
| `--requests`    | Total requests per test | `10000` | `1K - 100K`       |
| `--connections` | Concurrent connections  | `50`    | `10 - 200`        |
| `--batches`     | Number of test batches  | `3`     | `1 - 10`          |
| `--tool`        | Benchmark tool          | `wrk`   | `wrk` or `ab`     |

### 📊 Output Formats

The benchmark suite generates multiple output formats:

- **📈 CSV Data**: Raw performance metrics for analysis
- **📊 Summary Reports**: Aggregated results across scenarios
- **🔍 Detailed Logs**: Individual test execution details
- **📁 Organized Structure**: Date-based result directories

---

## 📚 Documentation

### 🏗️ Architecture Overview

This benchmark suite is designed with modularity, atomicity, and accuracy in mind:

```
go-web-benchmarks/
├── 🚀 cmd/              # Command-line interface
├── 🔧 internal/         # Core framework logic
│   ├── config/         # Configuration management
│   ├── progress/       # Progress tracking
│   ├── runner/         # Benchmark execution
│   └── types/          # Data structures
├── 🏗️ frameworks/       # Framework implementations
│   ├── flash/          # GoFlash implementation
│   ├── gin/            # Gin framework implementation
│   ├── gofiber/        # Fiber framework implementation
│   ├── echo/           # Echo framework implementation
│   └── chi/            # Chi framework implementation
├── 📊 results/         # Performance data and charts
├── ⚙️ config.yaml      # YAML configuration
└── 📋 README.md        # This documentation
```

### 🧪 Testing Methodology

Our approach ensures **fair and accurate comparisons**:

1. **Equivalent Implementations**: Each endpoint performs identical operations across frameworks
2. **Production Settings**: All servers run in optimized production mode
3. **Isolated Processes**: Frameworks run in separate processes to prevent interference
4. **Statistical Validity**: Multiple test batches ensure reliable results
5. **Resource Monitoring**: System resource usage tracked during tests
6. **Atomic Execution**: Tests are atomic and can be resumed from failures
7. **Deterministic Results**: Consistent execution environment and parameters

### 🔍 Interpreting Results

#### Key Metrics

- **RPS (Requests Per Second)**: Primary performance indicator
- **Latency Distribution**: Response time characteristics
- **Memory Usage**: Resource consumption patterns
- **CPU Utilization**: Processing efficiency

#### Performance Factors

- **Router Efficiency**: How quickly routes are matched and resolved
- **Middleware Overhead**: Processing cost of request/response pipeline
- **Memory Allocation**: Garbage collection and memory management impact
- **Serialization Speed**: JSON encoding/decoding performance

### 🚀 Advanced Features

#### 📊 Comprehensive Benchmark Examples

<details>
<summary><strong>🎯 Production-Level Load Testing Examples</strong></summary>

##### High-Volume Load Testing (1M Requests × 10 Batches)
```bash
# Ultimate stress test - 1 million requests per scenario, 10 statistical batches
go run ./cmd run --requests 1000000 --connections 100 --batches 10

# High-volume with all frameworks and scenarios (full comprehensive test)
go run ./cmd run --requests 1000000 --connections 200 --batches 10 --frameworks flash,gin,gofiber,echo,chi --scenarios simple,param,context,json,wildcard,groups,deepgroups,middleware,mw10

# Memory-intensive JSON processing test
go run ./cmd run --requests 500000 --connections 50 --batches 5 --scenarios json
```

##### Duration-Based Testing (1 Minute Per Test)
```bash
# 1-minute duration tests with statistical significance
go run ./cmd run --duration 1m --connections 50 --batches 3

# Extended duration testing for stability analysis
go run ./cmd run --duration 5m --connections 100 --batches 5

# Quick 1-minute validation across all scenarios
go run ./cmd run --duration 1m --connections 25 --batches 1 --scenarios simple,json,param
```

##### Scalability Testing
```bash
# Progressive connection scaling
go run ./cmd run --duration 30s --connections 10 --batches 3    # Light load
go run ./cmd run --duration 30s --connections 50 --batches 3    # Medium load  
go run ./cmd run --duration 30s --connections 200 --batches 3   # Heavy load
go run ./cmd run --duration 30s --connections 500 --batches 3   # Extreme load

# Framework comparison under different loads
go run ./cmd run --requests 100000 --connections 50 --frameworks flash,gin,gofiber
go run ./cmd run --requests 100000 --connections 200 --frameworks flash,gin,gofiber
```

</details>

#### Resume Capability

The benchmark suite supports resuming from failed runs:

```bash
# Resume from last failed run
./benchmark run --resume
```

#### Framework Filtering

Test specific frameworks only:

```bash
# Test only GoFlash and Gin
go run ./cmd run --frameworks flash,gin

# Compare top 3 performers
go run ./cmd run --duration 1m --frameworks flash,gin,gofiber --batches 5
```

#### Scenario Filtering

Test specific scenarios only:

```bash
# Test only simple and JSON scenarios
go run ./cmd run --scenarios simple,json

# Focus on API-heavy scenarios
go run ./cmd run --duration 1m --scenarios json,param,context --batches 3

# Test routing performance
go run ./cmd run --requests 50000 --scenarios simple,param,wildcard,groups,deepgroups
```

#### Custom Configuration

Override configuration parameters:

```bash
# Use ApacheBench instead of wrk
./benchmark run --tool ab

# Custom test duration
./benchmark run --duration 60s
```

---

## 🤝 Contributing

We welcome contributions to improve the benchmark suite! Here's how you can help:

### 🐛 Reporting Issues

- **Bug Reports**: Use the GitHub issue tracker
- **Feature Requests**: Suggest new frameworks or scenarios
- **Performance Issues**: Report unexpected results

### 🔧 Adding New Frameworks

1. **Create Framework Directory**: Add implementation in `frameworks/`
2. **Update Configuration**: Add framework to `config.yaml`
3. **Implement Endpoints**: Ensure all test scenarios are covered
4. **Test Thoroughly**: Run benchmarks to verify results

### 📊 Adding New Scenarios

1. **Define Scenario**: Add to `config.yaml` scenarios section
2. **Implement Handlers**: Add endpoints to all frameworks
3. **Update Documentation**: Document the new scenario
4. **Test Validation**: Ensure consistent behavior across frameworks

### 🧪 Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/config
go test ./internal/runner
```

### 📝 Code Style

- Follow Go conventions and best practices
- Add comprehensive documentation
- Include unit tests for new functionality
- Ensure atomic and deterministic behavior

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ for the Go community**

*Accurate, reproducible, and meaningful performance benchmarks*

</div>
