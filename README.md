# go-starter

A base Go service to start any backend on: layered architecture, Postgres, Redis
with distributed locks, S3, SQS consumers, structured logging, graceful shutdown,
and a full local stack via Docker Compose + LocalStack.

No example domains — the scaffolding is in place and empty. You add your domains
onto it.

One implementation of every backend: Postgres, Redis, S3, SQS. No in-memory or
on-disk stand-ins and no `driver:` switches. Locally these run as containers
(LocalStack provides S3 and SQS), so dev exercises the same code path as
production.

> **Working with an AI agent?** Point it at [AGENTS.md](./AGENTS.md) — it has the
> layering rules and copy-paste boilerplate for adding a domain.

## Quick start

Docker runs the infrastructure. The app runs on your machine.

```bash
make setup   # deps + mockgen
make infra   # postgres, redis, localstack in Docker
make run     # the app, on your host
```

`http://localhost:8080/health` returns `{"status":"ok"}`.

That split is deliberate: because the app is a normal local process, your IDE
debugger, breakpoints and rebuilds work with no extra setup. `make infra`
publishes every port to localhost, and `config/dev/config.yaml` points at
localhost, so there is nothing to reconfigure.

### Debugging

Run `make infra`, then debug `./cmd/server` from your IDE with:

- program / package: `./cmd/server`
- working directory: the repo root (`-config-path` is resolved relative to it)
- program arguments: `-config-path config/dev/config.yaml`

It's an ordinary local launch, so breakpoints, stepping and variable inspection
work as-is — no Delve server, no remote attach, no path mapping.

### Port conflicts

The containers publish 5432, 6379 and 4566. If something on your machine already
owns one of those, stop it, or change the port in two places: the `ports` entry in
`docker-compose.yaml` and the matching host in `config/dev/config.yaml`.

## Architecture

```
Router → Controller → Core → Repository → Postgres
                        └───→ Infrastructure (Redis / S3 / SQS)

SQS → Core (the service implements infrastructure.Consumer) → Repository
```

Strictly layered, manually dependency-injected. The whole object graph is built in
`cmd/server/main.go`.

```
cmd/server/              Entry point and dependency wiring
config/<env>/            Per-environment YAML config
internal/
  config/                Typed config + Viper loader
  constants/             Error codes, context keys, sentinel errors
  controller/            HTTP handlers + shared response helpers
  helpers/               Small pure utilities
  infrastructure/        Interfaces: cache.go, file_storage.go, events.go
    awsclient/           Shared aws.Config, S3 + SQS clients
    filestorage/         S3
    redis/               Cache + distributed lock
    sqs/                 Consumer pipeline + producer
  logger/                Zap wrapper
  middleware/            Request ID, request logging
  models/                GORM models + AutoMigrateModels registry
  repository/            DAO implementation, transactions
  repository.go          Repository / DAO / Lock interfaces
router/                  Route registration
scripts/localstack/      Bucket and queue bootstrap
```

You create `api/`, `internal/core/` and `internal/dto/` with your first domain.

## What you get

**Atomic transactions** — one transaction across multiple repository calls:

```go
err := repo.Atomic(ctx, func(store internal.DAO) error {
    order, err := store.CreateOrder(ctx, newOrder)
    if err != nil {
        return err
    }
    return store.DebitWallet(ctx, order.UserID, order.Amount)
})
```

**Distributed lock** — Redis + redsync; pass 0 for the 15 minute default TTL:

```go
err := repo.DistributedLock(ctx, "lock:user:"+id, 2*time.Minute, func() error {
    return doExclusiveWork(ctx, id)
})
// errors.Is(err, constants.LockAlreadyHeld) on contention
```

**Queue consumers** — a core service implements three methods
(`Unmarshal`, `ProcessData`, `GetTxnName`) and gets the long-poll → worker-pool →
commit pipeline, panic recovery, and exponential backoff for free.

**Object storage** — store, get, remove, and paginated list.

**Cache** — get, set with TTL, delete, exists.

**Graceful shutdown** — SIGINT/SIGTERM stops the consumers and drains in-flight
HTTP requests within `settings.server.shutdown_timeout`.

## Configuration

`config/<env>/config.yaml`, loaded via Viper. Any value can be overridden by an
environment variable: uppercase the path, dots to underscores, `APP_` prefix.

```
settings.sql.primary.password  ->  APP_SETTINGS_SQL_PRIMARY_PASSWORD
settings.aws.endpoint          ->  APP_SETTINGS_AWS_ENDPOINT
```

`settings.aws.endpoint` is the only value that differs between local and
production. Point it at LocalStack locally; leave it empty in real environments so
the AWS SDK resolves real endpoints and uses the default credential chain (IAM
role, env, shared config). Keep secrets out of YAML.

## Commands

```bash
make setup      # download deps, install mockgen
make infra      # start postgres + redis + localstack in Docker
make infra-down # stop them and drop volumes
make infra-logs # tail container logs
make build      # -> ./bin/server
make run        # build + run the app on the host
make dev        # make infra + make run
make test       # go test ./... -race -cover
make mocks      # regenerate mocks
make fmt vet    # format + vet
```

## Notes

- `settings.sql.auto_migrate` is `true` in dev. Use versioned migrations in real
  environments and set it to `false`.
- Consumer processing is at-least-once. Handlers must be idempotent.
- No authentication is wired into the router. Add auth middleware to the `v1` group
  before exposing this service to untrusted traffic.
