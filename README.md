# UE Session Manager

A Go-based session management module for 5G Core AMF (Access and Mobility Management Function) that provides caching and fast query capabilities for UE Context.

## Overview

This module is responsible for managing UE (User Equipment) sessions in a 5G Core network, providing:
- Temporary storage (caching) of UE Context
- Fast query capabilities for session information
- TTL-based session management
- Multi-index querying (IMSI, MSISDN, TMSI)

## Architecture

The project follows Clean Architecture principles with the following layers:

```
├── cmd/                    # Application entry points
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── database/          # Database connections
│   ├── domain/            # Business entities and interfaces
│   ├── handler/           # HTTP handlers
│   ├── middleware/        # HTTP middleware
│   ├── repository/        # Data access layer
│   └── service/           # Business logic
├── pkg/                   # Public packages
├── api/                   # API definitions
├── configs/               # Configuration files
├── scripts/               # Build and deployment scripts
├── docs/                  # Documentation
└── deployments/           # Deployment configurations
```

## Features

- **Session Management**: Create, read, update, delete UE sessions
- **TTL Support**: Automatic session expiration
- **Multi-Index Querying**: Query by IMSI, MSISDN, TMSI
- **REST API**: HTTP endpoints for session operations
- **Redis Backend**: High-performance caching with Redis
- **Clean Architecture**: Well-structured, testable code

## Getting Started

### Prerequisites

- Go 1.18+
- Redis 6.0+
- Docker (optional)

### Installation

1. Clone the repository
2. Install dependencies: `go mod tidy`
3. Start Redis server
4. Run the application: `go run cmd/server/main.go`

### Configuration

Copy `configs/config.example.yaml` to `configs/config.yaml` and adjust settings.

## API Endpoints

- `POST /sessions` - Create a new session
- `GET /sessions/:id` - Get session by TMSI
- `PUT /sessions/:id` - Update session
- `DELETE /sessions/:id` - Delete session
- `GET /sessions?imsi=...` - Query sessions by IMSI
- `GET /sessions?msisdn=...` - Query sessions by MSISDN

## Development

### Running Tests

```bash
go test ./...
go test -bench=. ./...
```

### Building

```bash
go build -o bin/sessionmgr cmd/server/main.go
```

## License

MIT License 