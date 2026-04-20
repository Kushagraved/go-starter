# go-starter

A production-ready Go REST API starter, mirroring the layered architecture of a real-world backoffice service.

## Architecture

```
Controller → Core (Business Logic) → Repository → Infrastructure
```

```
go-starter/
├── api/                    # Request/response JSON structs (by domain)
├── cmd/server/             # Main entry point
├── config/dev/             # Environment configs (yaml)
├── internal/
│   ├── config/             # Config loading
│   ├── constants/          # App-wide constants
│   ├── controller/         # HTTP handlers (Gin)
│   ├── core/               # Business logic (by domain)
│   ├── dto/                # Model → response transformers
│   ├── infrastructure/
│   │   ├── queue/          # In-memory queue (swap for SQS)
│   │   ├── redis/          # Redis client wrapper
│   │   └── storage/        # Local file storage (swap for S3)
│   ├── logger/             # Zap logger
│   ├── middleware/         # Request ID, logging
│   ├── models/             # Domain models (GORM)
│   └── repository/         # Data access (memory or postgres)
├── router/                 # Route registration
├── Dockerfile
├── docker-compose.yaml
└── Makefile
```

## Switching backends

| Concern  | Local (default)     | Production swap        |
|----------|---------------------|------------------------|
| Database | `db_driver: memory` | `db_driver: postgres`  |
| Queue    | `driver: memory`    | `driver: sqs`          |
| Storage  | `NewLocal()`        | implement `FileStorage` with S3 |
| Lock     | Redis (redsync)     | same — just point at real Redis |

## Quick start

```bash
# Spin everything up — app + postgres + redis
docker compose up

# API is available at http://localhost:8080
```

## Local build (without Docker)

```bash
make setup   # download deps
make build   # produces ./bin/server
make run     # build + run with config/dev/config.yaml
```

## API

| Method | Path           | Description     |
|--------|----------------|-----------------|
| GET    | /health        | Health check    |
| POST   | /v1/tasks      | Create task     |
| GET    | /v1/tasks      | List tasks      |
| GET    | /v1/tasks/:id  | Get task by ID  |
| PATCH  | /v1/tasks/:id  | Update task     |
| DELETE | /v1/tasks/:id  | Delete task     |

### Example

```bash
# Create
curl -X POST http://localhost:8080/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"my first task","description":"hello world"}'

# List
curl http://localhost:8080/v1/tasks

# Filter by status
curl "http://localhost:8080/v1/tasks?status=pending"
```

## Adding a new domain

1. Add model to `internal/models/`
2. Add interface to `internal/repository.go` (compose into `DAO`)
3. Implement in `internal/repository/` (memory + postgres variants)
4. Add service interface + impl in `internal/core/<domain>/`
5. Add API structs in `api/<domain>/`
6. Add DTO transformer in `internal/dto/`
7. Add controller methods in `internal/controller/`
8. Register routes in `router/<domain>_routes.go`
9. Wire service into `controller.Services` in `cmd/server/main.go`

## Generate mocks

```bash
mockgen -source=internal/repository.go \
        -destination=internal/repository/mock_repository.go \
        -package=repository
```
