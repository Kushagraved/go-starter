package queue

import "context"

// Message is the raw envelope passed through the queue.
type Message struct {
	Topic   string
	Payload []byte
}

// Consumer is implemented by any domain service that wants to process events.
// The queue infrastructure calls these methods — domain logic never touches the queue directly.
type Consumer interface {
	// Unmarshal deserialises raw bytes into a domain-specific struct.
	Unmarshal(ctx context.Context, msg Message) (interface{}, error)
	// ProcessData contains the business logic for handling the event.
	ProcessData(ctx context.Context, topic string, data interface{}) error
	// GetTxnName returns a label used for logging/tracing.
	GetTxnName() string
}

// ListenToEvents is the queue client interface.
// Swap the in-memory implementation for SQS/Kafka without touching domain code.
type ListenToEvents interface {
	StartConsumer(consumer Consumer, workerCount int)
}

// PushEvents is the producer interface.
type PushEvents interface {
	Produce(ctx context.Context, msg Message) error
}
