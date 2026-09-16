# Production image: build the binary, then ship it on a minimal base.
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o ./bin/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /app/bin/server ./bin/server
COPY --from=builder /app/config ./config

USER appuser

EXPOSE 8080

ENTRYPOINT ["./bin/server"]
CMD ["-config-path", "config/dev/config.yaml"]
