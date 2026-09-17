SOURCE = ./cmd/server/
OUTPUT = ./bin/server
CONFIG = config/dev/config.yaml

default: fmt build

setup:
	@echo "Downloading dependencies..."
	go mod download
	go install go.uber.org/mock/mockgen@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Done."

build:
	@echo "Building..."
	@mkdir -p bin
	go build -o $(OUTPUT) $(SOURCE)
	@echo "Build complete."

clean:
	rm -rf ./bin

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	ginkgo -r -race -cover

# ── Infrastructure in Docker ─────────────────────────────────────────────────
# Postgres, Redis and LocalStack (S3 + SQS). The app is not containerised; it runs
# on your machine so the debugger works normally.
infra:
	docker compose up -d

infra-logs:
	docker compose logs -f

infra-stop:
	docker compose stop

# Removes volumes too, so the database starts empty next time.
infra-down:
	docker compose down -v

# Apply the DDL schema to the local Postgres container. There is no auto-migration;
# this is how tables get created locally. Run after `make infra`.
db-schema:
	docker compose exec -T postgres psql -U postgres -d go_starter < scripts/db/schema.sql

# ── The app, on your machine ─────────────────────────────────────────────────
run: build
	./$(OUTPUT) -config-path $(CONFIG)

# Start infra if needed, then run.
dev: infra run

# Regenerate mocks. Run after changing any interface, and add a line per new
# interface file.
mocks:
	mockgen -source=internal/repository.go                  -destination=internal/repository_mock.go                  -package=internal
	mockgen -source=internal/infrastructure/cache.go        -destination=internal/infrastructure/cache_mock.go        -package=infrastructure
	mockgen -source=internal/infrastructure/file_storage.go -destination=internal/infrastructure/file_storage_mock.go -package=infrastructure
	mockgen -source=internal/infrastructure/events.go       -destination=internal/infrastructure/events_mock.go       -package=infrastructure
	mockgen -source=internal/core/expensesharing/expensesharing.go -destination=internal/core/expensesharing/expensesharing_mock.go -package=expensesharing

.PHONY: setup build clean fmt vet test infra infra-logs infra-stop infra-down db-schema run dev mocks
