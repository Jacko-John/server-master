# ServerMaster

`ServerMaster` is a Go-based proxy configuration management server designed to handle Clash subscriptions and rulesets. It provides a centralized service to merge local and remote proxy nodes, serve ruleset files, and perform background maintenance tasks like dynamic port mapping and rule auto-updates.

## Project Overview

- **Purpose:** A backend service for managing and generating Clash-compatible proxy configurations.
- **Main Technologies:**
  - **Language:** Go 1.26+
  - **Web Framework:** [Gin Gonic](https://github.com/gin-gonic/gin)
  - **Configuration:** YAML (using `gopkg.in/yaml.v3`)
  - **Task Scheduling:** [Cron](https://github.com/robfig/cron)
- **Architecture:**
  - `cmd/server`: Entry point and application bootstrapping.
  - `internal/app`: Lifecycle management (startup, shutdown, dependency injection).
  - `internal/api`: RESTful API handlers and routing.
  - `internal/service`: Core business logic (subscription generation, file management, port mapping).
  - `internal/model`: Data structures for Clash and Shadowsocks configurations.
  - `internal/config`: Configuration loading and validation.
  - `pkg/`: Reusable utilities (logger, data structures).
  - `workspace.d/`: Working directory for local proxy files and rulesets.

## Building and Running

### Build
To compile the project into an executable:
```bash
make build
# Or manually:
# CGO_ENABLED=0 go build -o ServerMaster ./cmd/server
```

### Run
To start the server with a specific configuration file:
```bash
./ServerMaster -c example/config.yaml
```

### Testing
To run all tests in the repository:
```bash
go test ./...
```

## API Endpoints

- **`GET /sub?token=xxx`**: Generates a unified Clash configuration by merging local proxy settings with remote subscriptions. Requires a valid token defined in the configuration.
- **`GET /file/:filename`**: Serves static ruleset files stored in the configured `rule-path`.

## Background Tasks (Cron)

The server runs several automated tasks based on the `config.yaml`:
1.  **Dynamic Port Mapping**: Periodically updates `iptables` rules to rotate listener ports for enhanced security.
2.  **Rule Set Auto-Update**: Periodically fetches updated rulesets from remote URLs and saves them to the local `rule-path`.

## Development Conventions

- **Clean Architecture**: Business logic is decoupled from HTTP handlers via service interfaces.
- **Graceful Shutdown**: The application listens for `SIGINT`/`SIGTERM` to ensure all services (HTTP server, Cron tasks) exit cleanly.
- **Structured Logging**: Uses Go's standard `log/slog` for structured, leveled logging.
- **Dependency Management**: Standard Go modules (`go.mod`).
