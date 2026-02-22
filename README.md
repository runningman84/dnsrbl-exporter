# dnsrbl-exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/runningman84/dnsrbl-exporter)](https://goreportcard.com/report/github.com/runningman84/dnsrbl-exporter)
[![Docker Pulls](https://img.shields.io/docker/pulls/runningman84/dnsrbl-exporter.svg)](https://hub.docker.com/r/runningman84/dnsrbl-exporter)
[![GitHub release](https://img.shields.io/github/release/runningman84/dnsrbl-exporter.svg)](https://github.com/runningman84/dnsrbl-exporter/releases)
[![License](https://img.shields.io/github/license/runningman84/dnsrbl-exporter.svg)](LICENSE)

## Introduction

A DNS-based Realtime Blacklist (DNSRBL) checker with a Prometheus metrics endpoint. Written in Go.

This tool allows you to monitor your IP addresses against multiple DNS blacklists and expose the results as Prometheus metrics for alerting and monitoring.

## Features

- 📊 Prometheus metrics endpoint
- 🔍 Support for multiple DNSRBL servers
- 🐳 Multi-architecture Docker images (amd64, arm64)
- ⚙️ Configurable via environment variables
- 🛠️ Includes `verify-lists` utility to test DNSRBL server responsiveness

## Building

### Build the main exporter

```sh
go build -o dnsrbl-exporter ./cmd/dnsrbl-exporter
```

### Build the verify-lists utility

```sh
go build -o verify-lists ./cmd/verify-lists
```

### Build all binaries

```sh
go build -o dnsrbl-exporter ./cmd/dnsrbl-exporter
go build -o verify-lists ./cmd/verify-lists
```

## Running Locally

### Run the exporter

```sh
./dnsrbl-exporter
```

Check version:
```sh
./dnsrbl-exporter -version
```

### Verify DNSRBL lists

The `verify-lists` utility tests all DNSRBL servers listed in `lists.txt` to ensure they are responsive:

```sh
./verify-lists
```

## Docker

### Pull the image

```sh
docker pull ghcr.io/runningman84/dnsrbl-exporter:latest
# or
docker pull runningman84/dnsrbl-exporter:latest
```

### Run the container

```sh
docker run -d -p 8000:8000 ghcr.io/runningman84/dnsrbl-exporter:latest
```

## Configuration

The container can be configured using these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DNSRBL_HTTP_BL_ACCESS_KEY` | API Key for https://www.projecthoneypot.org | None |
| `DNSRBL_DELAY_REQUESTS` | Sleep time between two subsequent requests (single list check) | 1 |
| `DNSRBL_DELAY_RUNS` | Sleep time between two subsequent runs (full list check) | 60 |
| `DNSRBL_LISTS` | Space separated list of RBLs (e.g., "dnsbl.httpbl.org zen.spamhaus.org") | None |
| `DNSRBL_LISTS_FILENAME` | Filename containing list of RBLs, one per line | lists.txt |
| `DNSRBL_CHECK_IP` | IP address to be checked (auto-discovery if not set) | None |
| `DNSRBL_PORT` | Listener port for metrics server | 8000 |

## Metrics

Prometheus metrics are exposed at `http://localhost:8000/metrics`

## Kubernetes / Helm

For Flux CD users, see the [flux/helm-release.yaml](flux/helm-release.yaml) file for a complete example configuration using the app-template chart with ServiceMonitor integration.

## Testing

Run the test suite:

```sh
go test ./...
```

Run tests with coverage:

```sh
go test -cover ./...
```

## License

See [LICENSE](LICENSE) file for details.
