# ServerMaster

`ServerMaster` is a Go-based proxy configuration management suite designed to handle Clash subscriptions and rulesets. It consists of a centralized server for managing and serving configurations and a client for local synchronization and process management.

## Project Overview

- **Purpose:** A backend service and client for managing and generating Clash-compatible proxy configurations.
- **Main Technologies:**
  - **Language:** Go 1.26+
  - **Web Framework:** [Gin Gonic](https://github.com/gin-gonic/gin) (Server)
  - **Configuration:** YAML (using `gopkg.in/yaml.v3`)
  - **Task Scheduling:** [Cron](https://github.com/robfig/cron) (Server)
- **Architecture:**
  - `cmd/server`: Entry point for the subscription management server.
  - `cmd/client`: Entry point for the local synchronization client.
  - `internal/app`: Server lifecycle management (startup, shutdown, dependency injection).
  - `internal/api`: Server RESTful API handlers and routing.
  - `internal/service`: Core business logic (subscription generation, file management, port mapping).
  - `internal/client`: Client logic for fetching, merging, and local configuration management.
  - `internal/model`: Data structures for Clash and Shadowsocks configurations.
  - `internal/config`: Configuration loading and validation.
  - `pkg/`: Reusable utilities (logger, data structures).
  - `workspace.d/`: Working directory for local proxy files, rulesets, and client data.

## Building and Running

### Build
To compile both components:
```bash
make build
# Or manually:
# CGO_ENABLED=0 go build -o ServerMaster ./cmd/server
# CGO_ENABLED=0 go build -o SMClient ./cmd/client
```

### Run Server
To start the server with a specific configuration file:
```bash
./ServerMaster -c example/config.yaml
```

### Run Client
The client can run as a one-shot sync or in daemon mode. The `server-url` in the client configuration should be the complete subscription URL (e.g., `http://1.2.3.4:8080/sub?token=xxx`):
```bash
# One-shot sync
./SMClient -c example/client.yaml

# Daemon mode (with background updates)
./SMClient -c example/client.yaml -d
```

### Testing
To run all tests in the repository:
```bash
go test ./...
```

## API Endpoints (Server)

- **`GET /sub?token=xxx`**: Generates a unified Clash configuration by merging local proxy settings with remote subscriptions. Requires a valid token defined in the configuration.
- **`GET /file/:filename`**: Serves static ruleset files stored in the configured `rule-path`.

## Background Tasks

### Server (Cron)
1.  **Dynamic Port Mapping**: Periodically updates `iptables` rules to rotate listener ports for enhanced security.
2.  **Rule Set Auto-Update**: Periodically fetches updated rulesets from remote URLs and saves them to the local `rule-path`.

### Client (Daemon)
1.  **Local Sync**: Periodically fetches configurations from `ServerMaster`, merges additional subscriptions, and applies local rules.
2.  **Mihomo Management**: Can automatically start and reload the Mihomo (Clash) process upon configuration updates.

## Development Conventions

- **Clean Architecture**: Business logic is decoupled from HTTP handlers and CLI entry points via service/client interfaces.
- **Graceful Shutdown**: Both server and client listen for `SIGINT`/`SIGTERM` to ensure all services (HTTP server, Cron tasks, Mihomo process) exit cleanly.
- **Structured Logging**: Uses Go's standard `log/slog` for structured, leveled logging.
- **Dependency Management**: Standard Go modules (`go.mod`).
