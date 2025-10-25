# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go-based Prometheus exporter that scrapes HomeKit accessory data from Homebridge APIs and exposes them as Prometheus metrics. The service authenticates with Homebridge, fetches all accessories and their service characteristics, converts numeric values to metrics, and serves them at `/metrics`.

## Build and Run Commands

### Local Development

```bash
# Build for current platform
go build

# Build for all supported platforms (Linux ARM64/ARM, macOS ARM64)
./build-all.sh

# Run locally (requires Homebridge instance)
go run . -u <username> -p <password> -uri http://localhost:8581

# Run with debug logging
go run . -u <username> -p <password> -debug

# Test the metrics endpoint
curl http://localhost:9123/metrics
```

### Docker

```bash
# Build the Docker image
docker build -t homebridge-exporter .

# Run with environment variables
docker run -d \
  -e HOMEBRIDGE_USERNAME=admin \
  -e HOMEBRIDGE_PASSWORD=yourpassword \
  -e HOMEBRIDGE_URI=http://homebridge:8581 \
  -p 9123:9123 \
  homebridge-exporter

# Run with debug logging
docker run -d \
  -e HOMEBRIDGE_USERNAME=admin \
  -e HOMEBRIDGE_PASSWORD=yourpassword \
  -e HOMEBRIDGE_URI=http://homebridge:8581 \
  -e HOMEBRIDGE_DEBUG=true \
  -p 9123:9123 \
  homebridge-exporter

# Build for specific platform (e.g., ARM64)
docker build --platform linux/arm64 -t homebridge-exporter:arm64 .
```

Note: There are currently no automated tests in this codebase.

## Architecture

### Core Components

The application has four main modules:

1. **main.go** - Entry point, HTTP server setup, graceful shutdown handling
2. **homebridge.go** - Session management and Homebridge API client
3. **auth.go** - Bearer token validation and application state management
4. **metrics.go** - Prometheus collector implementation

### Key Architectural Patterns

**Thread-Safe Session Management**: The `Session` struct in `homebridge.go` uses a mutex to protect token state. Token fetching is lazy (only happens when needed) and tokens are validated via `/api/auth/check` before use.

**Prometheus Collector Pattern**: Instead of rebuilding the registry on each scrape, `HomebridgeCollector` implements `prometheus.Collector` interface. The registry is created once at startup and stored in `AppState`. On each scrape, `Collect()` dynamically generates metrics by:
- Fetching current token (with automatic refresh if expired)
- Retrieving all accessories from Homebridge API
- Converting service characteristics to metrics
- Using `ConstMetric` for dynamic label values

**AppState Pattern**: The `AppState` struct aggregates all shared resources (session, config, authorization keys, Prometheus registry) and is passed to HTTP handlers, avoiding global state.

### Metric Generation Flow

1. Prometheus scrapes `/metrics`
2. `HomebridgeCollector.Collect()` is called
3. Token is fetched/refreshed via `Session.GetToken()` (thread-safe)
4. All accessories retrieved from `GET /api/accessories`
5. For each `ServiceCharacteristics`:
   - Skip if value is a string type (non-numeric)
   - Convert value to float64 using flexible type conversion
   - Generate metric name: `{prefix}_{service_type}_{characteristic_type}`
   - Emit gauge metric with service name as label
6. Metrics returned to Prometheus

### Data Models

- **Accessory**: Top-level device with metadata and array of `ServiceCharacteristics`
- **ServiceCharacteristics**: Individual characteristics like temperature, brightness, on/off state
- **Session**: Thread-safe wrapper around Homebridge authentication token
- **AppState**: Application-wide shared state container

### Important Implementation Details

**Type Conversion**: `convertToFloat64()` in `metrics.go` handles multiple types flexibly:
- Booleans → 0/1
- Numeric strings → parsed float64
- All numeric types → float64
- Rejects string values that can't be converted

**Metric Naming**: `toSnakeCase()` normalizes names to Prometheus conventions by inserting underscores before capitals, removing special characters, and deduplicating underscores.

**Authorization**: Optional bearer token validation via YAML keyfile (`authorization-keys.yml`). If the file doesn't exist, authorization is disabled (graceful degradation).

**OTP Handling**: The login function currently uses a hardcoded OTP value of "123" - this appears to be for test/default Homebridge instances.

## Configuration

Configuration can be provided via command-line flags or environment variables. Command-line flags take precedence over environment variables.

### Command-Line Flags

- `-u/-username`: Homebridge username (required)
- `-p/-password`: Homebridge password (required)
- `-uri`: Homebridge API endpoint (default: http://localhost:8581)
- `-port`: Metrics server port (default: 9123)
- `-prefix`: Metric name prefix (default: homebridge)
- `-keyfile`: Authorization keys YAML file (default: authorization-keys.yml)
- `-debug`: Enable verbose logging

### Environment Variables

| Environment Variable | Flag Equivalent | Default | Required |
|---------------------|-----------------|---------|----------|
| `HOMEBRIDGE_USERNAME` | `-u/-username` | - | Yes |
| `HOMEBRIDGE_PASSWORD` | `-p/-password` | - | Yes |
| `HOMEBRIDGE_URI` | `-uri` | http://localhost:8581 | No |
| `HOMEBRIDGE_PORT` | `-port` | 9123 | No |
| `HOMEBRIDGE_PREFIX` | `-prefix` | homebridge | No |
| `HOMEBRIDGE_KEYFILE` | `-keyfile` | authorization-keys.yml | No |
| `HOMEBRIDGE_DEBUG` | `-debug` | false | No |

**Note**: For `HOMEBRIDGE_DEBUG`, accepted values are: `true`, `1`, `yes` (case-sensitive)

## Development Notes

**No Test Suite**: There are no `*_test.go` files currently. When adding tests, consider testing:
- Token refresh logic in session management
- Type conversion edge cases in `convertToFloat64()`
- Metric name normalization in `toSnakeCase()`
- Collector behavior with mocked Homebridge responses

**Graceful Shutdown**: The server implements proper shutdown with a 30-second timeout, cancellation context propagation, and WaitGroup coordination. Maintain this pattern when adding new goroutines.

**Error Handling Philosophy**: The metrics collector logs errors but doesn't fail - it skips problematic values and continues. This ensures partial failures don't break the entire metrics endpoint.
