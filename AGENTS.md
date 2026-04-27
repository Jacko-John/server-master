# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ServerMaster is a Go-based proxy configuration management suite for handling Clash subscriptions and rulesets. It consists of:
- **ServerMaster** (`cmd/server`): Centralized server for managing and serving proxy configurations
- **SMClient** (`cmd/client`): Local client for synchronization and Mihomo (Clash) process management

## Build and Development Commands

```bash
# Build both components
make build
# Or individually:
make build-server  # Builds ServerMaster binary
make build-client  # Builds SMClient binary

# Run all tests
go test ./...

# Run specific package tests
go test ./internal/service/...
go test -v ./internal/service/...  # Verbose output

# Run single test
go test -run TestGenerateConfig ./internal/service/
```

## Running the Application

### Server
```bash
./ServerMaster -c example/config.yaml
```

### Client
```bash
# One-shot sync
./SMClient -c example/client.yaml

# Daemon mode (with background updates)
./SMClient -c example/client.yaml -d
```

## Architecture

### Layered Structure

```
cmd/           → Entry points (CLI flags, signal handling)
internal/app/  → Application lifecycle (start/stop, dependency injection)
internal/api/  → HTTP handlers and routing (Gin framework)
internal/service/ → Business logic (subscription generation, file management, cron tasks)
internal/model/ → Data structures (Clash/Shadowsocks config models)
internal/config/ → Configuration loading and validation
internal/client/ → Client-side logic (syncer, Mihomo manager)
pkg/           → Reusable utilities (logger, data structures)
```

### Key Components

**Server (`internal/app/`)**:
- `app.go`: Main application lifecycle management, initializes services, starts cron and HTTP server
- Service Container pattern: All business services managed through `service.Container`

**API Layer (`internal/api/`)**:
- Router interface for modular route registration
- `sub.go`: Handles `/sub?token=xxx` endpoint for subscription generation
- `file.go`: Handles `/file/:filename` for serving static ruleset files

**Service Layer (`internal/service/`)**:
- `subscription.go`: Merges base proxy config with external subscriptions (additions), applies port randomization
- `ruleset.go`: Fetches and caches remote rulesets to local `rule-path`
- `port.go`: Dynamic port mapping via iptables for security
- `file.go`: Static file serving
- `cron.go`: Generic task scheduler - tasks implement `Task` interface with optional `Init()` and `Cleanup()`
- `container.go`: Dependency injection container for all services

**Client (`internal/client/`)**:
- `syncer.go`: Fetches config from ServerMaster, merges additional subscriptions, saves locally
- `mihomo/manager.go`: Process supervisor for Mihomo kernel with auto-restart on failure

**Models (`internal/model/`)**:
- `clash.go`: Clash configuration structures (proxies, proxy-groups, rules)
- `ss.go`: Shadowsocks URL parsing

**Utilities (`pkg/`)**:
- `logger/`: Wrapper around `log/slog` with JSON/text format support
- `utils/`: Thread-safe data structures (SafeMap, Set, Queue)

### Caching Strategy

- Base config (proxy.yaml): Cached by file modification time
- External dependencies: Cached with 5-minute TTL
- Shared `utils.Queue[string]` for dynamic port randomization

### Cron Tasks

Server runs two optional background tasks:
1. **Dynamic Port**: Randomizes proxy listener ports via iptables (configurable via `cron.dynamic-port`)
2. **Rule Set**: Downloads rulesets from remote URLs (configurable via `cron.rule-set`)

### Configuration Files

- **Server config**: `tokens` (required), `proxy-path`, `rule-path`, `additions` (external subscriptions)
- **Client config**: `server-url` (full subscription URL), `additions`, `mihomo` section for process management
- **workspace.d/**: Working directory containing proxy configs, rulesets, and client data

## Development Conventions

- **Graceful Shutdown**: Both server and client handle `SIGINT`/`SIGTERM` for clean exit
- **Structured Logging**: Uses `log/slog` throughout
- **Error Groups**: Uses `golang.org/x/sync/errgroup` for concurrent operations with cancellation
- **Context Passing**: All blocking operations accept `context.Context`
- **Interface-Based Design**: Cron tasks implement `Task` interface, API routes implement `Router` interface

## Testing Notes

- Tests use standard Go testing with table-driven patterns
- Mock files in `example/` directory for config testing
- Service layer has comprehensive test coverage for config generation, file serving, and ruleset logic
