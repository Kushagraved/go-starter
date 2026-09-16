package infrastructure

import (
	"context"
	"time"

	"go-starter/internal/logger"
)

/*
Consumer is the contract a queue consumer implements — three methods, no queue
plumbing. Implement it directly on the core service that owns the domain rather
than wrapping the service in a separate consumer type.

Hand the service to ListenToEvents.StartConsumer and the polling, worker pool,
panic recovery and offset committing are provided.
*/
type Consumer interface {
	// Unmarshal turns raw message bytes into whatever ProcessData expects.
	Unmarshal(ctx context.Context, metadata QueueMessageMetadata, loggerObj logger.Logger) (interface{}, error)
	// ProcessData holds the business logic. Returning an error leaves the message
	// on the queue to be retried, so implementations must be idempotent.
	ProcessData(ctx context.Context, loggerObj logger.Logger, topic string, data interface{}, retryCount int64) error
	// GetTxnName names the consumer in logs.
	GetTxnName() string
}

// QueueMessageMetadata is handed to Consumer.Unmarshal.
type QueueMessageMetadata struct {
	Data           []byte
	Topic          string
	CommitMetadata interface{}
}

// EventResponse is a raw message polled off a queue.
type EventResponse struct {
	RawData        []byte
	QueueName      string
	CommitMetadata interface{}
	RetryCount     int64
}

// ListenToEvents configures and runs a consumer.
type ListenToEvents interface {
	WithMaxNumberOfMessages(maxMessages int32) ListenToEvents
	WithVisibilityTimeout(visibilityTimeout int32) ListenToEvents
	WithWaitTimeSeconds(waitTime int32) ListenToEvents
	WithExponentialBackoff() ListenToEvents
	// StartConsumer returns immediately; the pipeline runs until ctx is cancelled.
	StartConsumer(ctx context.Context, consumer Consumer, workerCount int) bool
}

// Message is an outbound event.
type Message struct {
	Topic string
	Data  []byte
	// DeliveryDelay postpones visibility. Capped at 15 minutes by SQS.
	DeliveryDelay time.Duration
}

// PushEvents publishes events onto a queue.
type PushEvents interface {
	Produce(ctx context.Context, event *Message) error
}
