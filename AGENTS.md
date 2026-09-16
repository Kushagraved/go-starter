# AGENTS.md

Guidance for AI agents and humans working in this repository. Read this before
writing code: it defines where each kind of code belongs and the conventions to
follow.

## What this is

A base Go service: layering, config, Postgres, Redis + distributed lock, S3, SQS
consumers/producers, structured logging, graceful shutdown, and a local stack via
Docker Compose + LocalStack.

**There are no example domains.** The scaffolding is in place and empty on purpose
— you add domains onto it. Everything here is deliberately minimal; extend an
interface when you actually need the method rather than up front.

There is one implementation of every backend: Postgres for the database, Redis for
cache and locks, S3 for objects, SQS for queues. No in-memory or on-disk
stand-ins, no `driver:` switches. Locally these run as containers (LocalStack
provides S3 and SQS), so dev exercises the same code path as production.

## Stack

| Concern    | Choice                              |
|------------|-------------------------------------|
| HTTP       | Gin                                 |
| Database   | Postgres via GORM                   |
| Config     | Viper (YAML + `APP_` env overrides)  |
| Logging    | Zap via `internal/logger`            |
| Cache/Lock | Redis (`go-redis`) + `redsync`       |
| Object     | S3 (`aws-sdk-go-v2`)                 |
| Queue      | SQS (`aws-sdk-go-v2`)                |
| Local AWS  | LocalStack                          |
| Mocks      | `mockgen` (`go.uber.org/mock`)       |
| Tests      | Ginkgo + Gomega (`github.com/onsi`)  |
| Schema     | Hand-written DDL (no auto-migrate)   |
| IDs        | Postgres `bigserial` (`int64`)       |

---

## Architecture

```
Router → Controller → Core → Repository → Postgres
                        └───→ Infrastructure (Redis / S3 / SQS)

SQS → Core (the service implements infrastructure.Consumer) → Repository
```

### Non-negotiable rules

1. **Never skip a layer.** A controller may not touch the repository. Core may not
   import `gin`. `internal/models` holds domain entities and is shared by core and
   repository; the **database row structs are private to `internal/repository`**
   and never leave it (convert to `models.*` via `ToEntity`).
2. **Core holds all business logic.** Controllers bind and shape. Repositories read
   and write. Everything else is core.
3. **Interface one level above the implementation.** `internal/repository.go`
   declares the DAO, `internal/repository/` implements it.
   `internal/infrastructure/*.go` declares, `internal/infrastructure/<impl>/`
   implements.
4. **`context.Context` first, always propagated.** Never `context.Background()`
   mid-chain.
5. **Wire everything in `cmd/server/main.go`.** No globals or service locators.
6. **A core service implements `infrastructure.Consumer` itself.** Do not create a
   separate consumer struct that wraps the service.
7. **Never hand-edit `*_mock.go`.** Regenerate with `make mocks`.

---

## Directory map — where to put what

```
api/<domain>/            JSON request/response structs. Tags only, no logic.
cmd/server/main.go       Entry point. The whole object graph is built here.
config/<env>/config.yaml Per-environment config.
internal/
  config/                Typed config + Viper loader.
  constants/             Error codes, context keys, sentinel errors.
  controller/            HTTP handlers. server.go has Services + shared helpers.
  core/<domain>/         Business logic. Service interface + implementation.
  dto/                   models -> api response mapping.
  helpers/               Small pure utilities (app.StackTrace).
  infrastructure/        Interfaces: cache.go, file_storage.go, events.go
    awsclient/           Shared aws.Config, S3 + SQS clients.
    filestorage/         S3 implementation.
    redis/               Cache + distributed lock implementation.
    sqs/                 Consumer pipeline + producer.
  logger/                Zap wrapper and field constructors.
  middleware/            Request ID, request logging.
  models/                Domain entities: plain structs, no GORM tags, int64 IDs.
  repository/            DAO implementation: repository.go (client), dao.go
                         (Atomic, lock), constants.go (TableNames enum),
                         structs.go (private DB rows + ToEntity), <domain>.go.
  repository.go          Repository / DAO / Lock interfaces.
router/                  Route registration, one file per domain.
scripts/db/schema.sql    Hand-written DDL — the schema of record, no auto-migrate.
scripts/localstack/      Bucket and queue bootstrap for local runs.
```

`api/`, `internal/core/` and `internal/dto/` do not exist yet — create them with
your first domain.

---

## Adding a domain — full boilerplate

Work in this order; each step compiles before the next.

### 1. Domain entity — `internal/models/user.go`

`models` holds **domain entities only** — plain Go structs, no GORM tags, no
`TableName()`. The core layer speaks these types. The database representation is
private to the repository (see step 3).

```go
package models

import "time"

type User struct {
    ID        int64     // Postgres bigserial, assigned by the DB
    Email     string
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**IDs are Postgres auto-increment (`bigserial`), not UUIDs.** The ID is `int64`,
zero until the row is inserted; the database assigns it. Do not generate IDs in
Go.

There is **no auto-migration.** Add the table to the DDL in `scripts/db/schema.sql`
instead (see step 1b).

### 1b. Schema — `scripts/db/schema.sql`

The schema is owned by explicit DDL, applied by hand or by the local bootstrap.
Add a `CREATE TABLE` per model, with `BIGSERIAL PRIMARY KEY` and the unique
indexes the domain needs.

```sql
CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_users_email UNIQUE (email)
);
```

Name unique constraints explicitly (e.g. `uq_users_email`) so the repository can
map a violation back to a specific domain error.

### 2. Repository interface — `internal/repository.go`

```go
type DAO interface {
    Atomic(ctx context.Context, fn func(store DAO) error) error
    UserRepository
}

type UserRepository interface {
    CreateUser(ctx context.Context, user models.User) (models.User, error)
    GetUserByID(ctx context.Context, id int64) (models.User, error)
    ListUsers(ctx context.Context) ([]models.User, error)
}
```

Add only the methods you actually call. Do not add generic filter/list variants
speculatively.

### 3. Repository implementation — private DB structs

The database shape is **private to the repository package.** For each table,
declare a lowercase row struct with GORM column tags in
`internal/repository/structs.go`, plus a `ToEntity()` that maps it to the
`models` domain entity and a `from*` builder for the reverse. The row struct never
leaves the package — every method returns `models.*`.

`internal/repository/structs.go`:

```go
package repository

import (
    "time"

    "go-starter/internal/models"
)

type userRow struct {
    ID        int64     `gorm:"column:id"` // bigserial; DB assigns on insert
    Email     string    `gorm:"column:email"`
    Name      string    `gorm:"column:name"`
    CreatedAt time.Time `gorm:"column:created_at"`
    UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (r userRow) ToEntity() models.User {
    return models.User{
        ID: r.ID, Email: r.Email, Name: r.Name,
        CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
    }
}

// fromUser omits ID so Postgres assigns the bigserial value.
func fromUser(u models.User) userRow {
    return userRow{Email: u.Email, Name: u.Name}
}
```

Table names live in a `TableNames` enum in `internal/repository/constants.go` so a
typo is a compile error:

```go
type TableNames string

const Users TableNames = "users"

func (t TableNames) ToString() string { return string(t) }
```

`internal/repository/user.go` — methods hang off `client` so `Atomic` scopes them,
query through `.Table(...)`, and translate GORM/driver errors into sentinels so
upper layers never import GORM:

```go
package repository

func (c client) CreateUser(ctx context.Context, user models.User) (models.User, error) {
    row := fromUser(user)
    if err := c.db.WithContext(ctx).Table(Users.ToString()).Create(&row).Error; err != nil {
        return models.User{}, translateUserWriteErr(err) // maps uq_users_email -> DuplicateEmail
    }
    return row.ToEntity(), nil // row.ID now holds the DB-assigned bigserial
}

func (c client) GetUserByID(ctx context.Context, id int64) (models.User, error) {
    var row userRow
    err := c.db.WithContext(ctx).Table(Users.ToString()).First(&row, "id = ?", id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return models.User{}, constants.RecordDoesntExists
    }
    if err != nil {
        return models.User{}, err
    }
    return row.ToEntity(), nil
}

func (c client) ListUsers(ctx context.Context) ([]models.User, error) {
    var rows []userRow
    if err := c.db.WithContext(ctx).Table(Users.ToString()).Find(&rows).Error; err != nil {
        return nil, err
    }
    users := make([]models.User, 0, len(rows))
    for _, row := range rows {
        users = append(users, row.ToEntity())
    }
    return users, nil
}
```

### 4. Core service — `internal/core/user/user.go`

```go
package user

type Service interface {
    Create(ctx context.Context, req CreateRequest) (models.User, error)
    Get(ctx context.Context, id int64) (models.User, error)
}

type CreateRequest struct {
    Email string
    Name  string
}

type service struct {
    repo     internal.Repository
    producer infrastructure.PushEvents
    logger   logger.Logger
}

func NewService(repo internal.Repository, producer infrastructure.PushEvents, loggerObj logger.Logger) Service {
    return &service{repo: repo, producer: producer, logger: loggerObj}
}

func (s *service) Create(ctx context.Context, req CreateRequest) (models.User, error) {
    if req.Email == "" {
        return models.User{}, errors.New("email is required")
    }
    // No ID generated here: it is a Postgres bigserial the DB assigns on insert.
    return s.repo.CreateUser(ctx, models.User{
        Email: req.Email,
        Name:  req.Name,
    })
}

func (s *service) Get(ctx context.Context, id int64) (models.User, error) {
    return s.repo.GetUserByID(ctx, id)
}
```

### 5. API structs — `api/user/user.go`

```go
package user

type CreateUserRequest struct {
    Email string `json:"email" binding:"required,email"`
    Name  string `json:"name"  binding:"required"`
}

type UserResponse struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}
```

### 6. DTO — `internal/dto/user.go`

```go
package dto

func ToUserResponse(user models.User) apiuser.UserResponse {
    return apiuser.UserResponse{ID: user.ID, Email: user.Email, Name: user.Name}
}
```

### 7. Controller — `internal/controller/user.go`

```go
package controller

func (s *Server) CreateUser(ctx *gin.Context) {
    var req apiuser.CreateUserRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
        return
    }

    created, err := s.services.User.Create(ctx.Request.Context(), user.CreateRequest{
        Email: req.Email,
        Name:  req.Name,
    })
    if err != nil {
        s.RespondError(ctx, "CreateUser", err)
        return
    }

    ctx.JSON(http.StatusCreated, dto.ToUserResponse(created))
}

func (s *Server) GetUser(ctx *gin.Context) {
    found, err := s.services.User.Get(ctx.Request.Context(), ctx.Param("id"))
    if err != nil {
        s.RespondError(ctx, "GetUser", err)
        return
    }
    ctx.JSON(http.StatusOK, dto.ToUserResponse(found))
}
```

`RespondError` already maps `RecordDoesntExists` to 404 and `LockAlreadyHeld` to
409, and logs anything unexpected as a 500. Use it instead of hand-writing status
codes.

Add the service to `Services` in `internal/controller/server.go`:

```go
type Services struct {
    User user.Service
}
```

### 8. Routes — `router/user_routes.go`

```go
package router

func registerUserRoutes(group *gin.RouterGroup, srv *controller.Server) {
    users := group.Group("/users")
    {
        users.POST("", srv.CreateUser)
        users.GET("/:id", srv.GetUser)
    }
}
```

Register it in `router.Init`:

```go
v1 := engine.Group("/v1")
registerUserRoutes(v1, srv)
```

### 9. Wire it — `cmd/server/main.go`

```go
userService := user.NewService(repo, producer, appLogger)
services := controller.Services{User: userService}
```

### 10. Add the table to the schema

Add the `CREATE TABLE` to `scripts/db/schema.sql` (step 1b). There is no
auto-migration to pick it up.

### 11. Finish

```bash
make mocks && make fmt && make build && make test
```

---

## Testing — Ginkgo + Gomega

Tests use [Ginkgo](https://onsi.github.io/ginkgo/) with Gomega matchers, run via
`ginkgo -r` (see `make test`). Each package under test gets a suite bootstrap file
plus one spec file per unit.

Suite bootstrap — `internal/core/user/user_suite_test.go`:

```go
package user_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestUser(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "User Core Suite")
}
```

Spec — `internal/core/user/user_test.go`. Build the mocked repository in
`BeforeEach`, exercise the service in `It`, assert with Gomega:

```go
package user_test

import (
    "context"

    "go-starter/internal"
    "go-starter/internal/constants"
    "go-starter/internal/core/user"
    "go-starter/internal/models"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "go.uber.org/mock/gomock"
)

var _ = Describe("Create", func() {
    var (
        ctrl *gomock.Controller
        repo *internal.MockRepository
        svc  user.Service
    )

    BeforeEach(func() {
        ctrl = gomock.NewController(GinkgoT())
        repo = internal.NewMockRepository(ctrl)
        svc = user.NewService(repo, nil, nil)
    })

    AfterEach(func() { ctrl.Finish() })

    It("returns DuplicateEmail when the email is taken", func() {
        repo.EXPECT().
            CreateUser(gomock.Any(), gomock.Any()).
            Return(models.User{}, constants.DuplicateEmail)

        _, err := svc.Create(context.Background(), user.CreateRequest{
            Email: "taken@example.com", Name: "Alice",
        })
        Expect(err).To(MatchError(constants.DuplicateEmail))
    })
})
```

Keep tests table-driven with `DescribeTable`/`Entry` where the same assertion runs
over many inputs. Mock the repository interface (`make mocks`) — never hit a real
database in a unit test.

---

## The utilities you get

### Atomic transactions

Every call through `store` runs in one transaction. Returning an error rolls back;
a panic rolls back and becomes an error.

```go
err := s.repo.Atomic(ctx, func(store internal.DAO) error {
    order, err := store.CreateOrder(ctx, newOrder)
    if err != nil {
        return err
    }
    return store.DebitWallet(ctx, order.UserID, order.Amount)
})
```

Use `store` inside the closure, never `s.repo` — that would run outside the
transaction.

### Distributed lock

```go
err := s.repo.DistributedLock(ctx, "lock:user:"+id, 2*time.Minute, func() error {
    return s.doExclusiveWork(ctx, id)
})
if errors.Is(err, constants.LockAlreadyHeld) {
    // someone else holds it — decide whether that is an error or a no-op
}
```

Pass `0` for the default 15 minute TTL. Contention returns
`constants.LockAlreadyHeld`.

### Queue consumer

The core service implements the three `infrastructure.Consumer` methods directly:

```go
// internal/core/user/consumer.go
func (s *service) GetTxnName() string { return "UserEventsConsumer" }

func (s *service) Unmarshal(_ context.Context, metadata infrastructure.QueueMessageMetadata,
    _ logger.Logger) (interface{}, error) {
    var event models.UserEvent
    if err := json.Unmarshal(metadata.Data, &event); err != nil {
        return nil, errors.Wrap(err, "invalid user event payload")
    }
    return &event, nil
}

func (s *service) ProcessData(ctx context.Context, loggerObj logger.Logger, topic string,
    data interface{}, retryCount int64) error {
    event, ok := data.(*models.UserEvent)
    if !ok {
        return errors.Errorf("unexpected payload type %T", data)
    }
    return s.handleEvent(ctx, event)
}
```

Wire it up in three places.

1. Add a named queue field to `config.SQSCfg` — queues are static fields, not a
   map, so a typo is a compile error rather than a silent no-op:

```go
type SQSCfg struct {
    Enable     bool `yaml:"enable"`
    Workers    int  `yaml:"workers"`
    AutoCommit bool `yaml:"auto_commit"`

    TestEvents QueueCfg `yaml:"test_events"`
    UserEvents QueueCfg `yaml:"user_events"`
}
```

2. Add the matching block to `config/dev/config.yaml` under `settings.sqs`, and
   create the queue in `scripts/localstack/init-aws.sh`:

```yaml
    user_events:
      enable: true
      queue_name: "user-events"
      workers: 2
      max_messages: 10
      visibility_timeout: 30
      wait_time_seconds: 20
      require_exp_backoff: true
```

3. Add one line to `startConsumers` in `cmd/server/main.go`. It already receives
   `controller.Services`, so the service is reached through it — no extra
   parameter, and HTTP and queue traffic enter through the same object:

```go
startConsumer(ctx, awsClients, appLogger, sqsCfg, sqsCfg.UserEvents, services.User)
```

A sample queue (`test_events` → `test-events`) already ships in config and is
created in LocalStack, disabled. Point a `startConsumer` line at it to try the
pipeline end to end:

```bash
docker compose exec localstack awslocal sqs send-message \
  --queue-url "$(docker compose exec -T localstack awslocal sqs get-queue-url --queue-name test-events --output text)" \
  --message-body '{"hello":"world"}'
```

Semantics that matter:

- `auto_commit: false` deletes a message only after `ProcessData` returns nil.
  Returning an error leaves it to be retried, so **handlers must be idempotent**.
- `require_exp_backoff: true` extends the visibility timeout by
  `timeout × 2 × receiveCount` on failure instead of hot-looping.
- `sqs.enable: false` is the master switch and starts no consumers at all.

Publishing:

```go
err := s.producer.Produce(ctx, &infrastructure.Message{
    Topic:         "user-events",
    Data:          payload,
    DeliveryDelay: 30 * time.Second, // optional, max 15 min
})
```

A publish failure after a committed write is a real tradeoff: either log and move
on (the write already happened), or use an outbox table inside `Atomic` if the
event and the write must be atomic.

### Cache

```go
err := cache.Set(ctx, "user:"+id, payload, 5*time.Minute)
value, err := cache.Get(ctx, "user:"+id) // constants.RecordDoesntExists when absent
```

### Object storage

```go
path, err := fileStorage.StoreFile(ctx, bucket, "invoices/inv-1.pdf", data,
    &infrastructure.FileStorageParams{ContentType: "application/pdf"})

data, err := fileStorage.GetFile(ctx, bucket, path) // RecordDoesntExists when absent
objects, err := fileStorage.ListObjects(ctx, bucket, "invoices/")
```

---

## Logging

Structured, always via `internal/logger`. Each call takes a bucket (component) and
handler (function) so logs are greppable. Request ID and trace ID are attached
automatically from the context.

```go
s.logger.Error(ctx, "UserService", "Create", "failed to create user",
    logger.NewString("email", req.Email), logger.NewError(err))
```

Use the `logger.New*` constructors. Never import `zap` outside the logger package,
and never use `fmt` for operational output.

---

## Configuration

`config/<env>/config.yaml` via Viper. Any value can be overridden by an env var:
uppercase the path, dots to underscores, `APP_` prefix.

```
settings.sql.primary.password  ->  APP_SETTINGS_SQL_PRIMARY_PASSWORD
settings.aws.endpoint          ->  APP_SETTINGS_AWS_ENDPOINT
```

Keep secrets out of YAML — inject them as env vars. `config.Load` validates
required fields and fails at boot rather than at first use.

`settings.aws.endpoint` is the only value that differs between local and
production: LocalStack's address locally, empty in real environments.

---

## Commands

```bash
make setup      # download deps, install mockgen
make infra      # start postgres + redis + localstack in Docker
make infra-down # stop them and drop volumes
make build      # -> ./bin/server
make run        # build + run the app on the host
make dev        # make infra + make run
make test       # ginkgo -r -race -cover
make mocks      # regenerate mocks
make fmt vet    # format + vet
```

Docker holds infrastructure only; the app always runs as a host process, which is
what keeps IDE debugging trivial. `config/dev/config.yaml` therefore points at
`localhost`, not at compose service names.

Do not run servers or watchers in a blocking foreground call during an automated
session; use `make build` and `make test` to verify.

## Checklist before finishing

- [ ] No layer skipped; `gin` only in controller/middleware/router
- [ ] `context.Context` first parameter, propagated everywhere
- [ ] Errors wrapped with context; sentinels compared with `errors.Is`
- [ ] `constants.RecordDoesntExists` returned for missing rows, not raw GORM errors
- [ ] `models` entities are plain structs (no GORM tags); DB rows stay private in
      `repository` and convert via `ToEntity`
- [ ] IDs are Postgres `bigserial` (`int64`), assigned by the DB — never generated in Go
- [ ] New table added to `scripts/db/schema.sql` (no auto-migrate)
- [ ] Structured logging with bucket + handler, no `fmt`
- [ ] Multi-write flows inside `Atomic`; cross-instance critical sections locked
- [ ] Consumer handlers idempotent
- [ ] New/changed interfaces re-mocked (`make mocks`) and added to the Makefile
- [ ] Ginkgo specs added/updated; `make fmt vet build test` clean
