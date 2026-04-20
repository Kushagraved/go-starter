package queue

import "context"

// Message is a generic event envelope.
type Message struct {
	Queue   string
	Payload []byte
}

// Producer publishes messages to a queue.
type Producer interface {
	Publish(ctx context.Context, msg Message) error
}

// Consumer processes messages from a queue.
type Consumer interface {
	Subscribe(ctx context.Context, queueName string, handler func(ctx context.Context, msg Message) error) error
}
