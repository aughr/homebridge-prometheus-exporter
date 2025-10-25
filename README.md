# Prometheus Exporter for Homebridge
A simple exporter for Prometheus that reads information about all your devices and exports all values as Prometheus metrics.

## Usage

Configuration can be provided via command-line flags or environment variables. Command-line flags take precedence over environment variables.

### Command-Line Flags

```text
USAGE:
    homebridge-exporter [OPTIONS]

OPTIONS:
    -u, -username <USERNAME>    Homebridge username (required)
    -p, -password <PASSWORD>    Homebridge password (required)
    -uri <URI>                  Homebridge UI uri [default: http://localhost:8581]
    -port <PORT>                Metrics webserver port [default: 9123]
    -prefix <PREFIX>            Registry metrics prefix [default: homebridge]
    -keyfile <KEYFILE>          Authorization keys file [default: authorization-keys.yml]
    -debug                      Debug mode (displays additional log lines)

EXAMPLES:
    homebridge-exporter -u admin -p mypassword -uri http://homebridge:8581
    homebridge-exporter -username admin -password mypassword -debug
```

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

This software scrapes all the accessories from Homebridge APIs and creates Prometheus metrics out of all services information.
All the metrics are then exposed under the standard path `/metrics` by the embedded HTTP server.

## Build from source

### Local Build

```bash
# Build for current platform
go build

# Build for all supported platforms (Linux ARM64/ARM, macOS ARM64)
./build-all.sh
```

### Docker Build

```bash
# Build the Docker image
docker build -t homebridge-exporter .

# Build for specific platform (e.g., ARM64)
docker build --platform linux/arm64 -t homebridge-exporter:arm64 .
```

## Running with Docker

```bash
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

# Using docker-compose
version: '3.8'
services:
  homebridge-exporter:
    image: homebridge-exporter
    environment:
      HOMEBRIDGE_USERNAME: admin
      HOMEBRIDGE_PASSWORD: yourpassword
      HOMEBRIDGE_URI: http://homebridge:8581
      HOMEBRIDGE_DEBUG: "false"
    ports:
      - "9123:9123"
    restart: unless-stopped
```

