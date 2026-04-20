FROM golang:1.18-alpine

RUN apk add --no-cache bash make git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

EXPOSE 8080

CMD ["./bin/server", "-config-path", "config/dev/config.yaml"]
