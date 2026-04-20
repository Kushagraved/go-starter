package queue

import (
	"context"
	"sync"
)

// memoryQueue is an in-process channel-based queue — drop-in for SQS in local dev.
type memoryQueue struct {
	mu       sync.RWMutex
	channels map[string]chan Message
	bufSize  int
}

func NewMemoryQueue(bufSize int) interface {
	Producer
	Consumer
} {
	return &memoryQueue{
		channels: make(map[string]chan Message),
		bufSize:  bufSize,
	}
}

func (m *memoryQueue) channel(name string) chan Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.channels[name]; !ok {
		m.channels[name] = make(chan Message, m.bufSize)
	}
	return m.channels[name]
}

func (m *memoryQueue) Publish(_ context.Context, msg Message) error {
	ch := m.channel(msg.Queue)
	ch <- msg
	return nil
}

func (m *memoryQueue) Subscribe(ctx context.Context, queueName string, handler func(context.Context, Message) error) error {
	ch := m.channel(queueName)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				_ = handler(ctx, msg)
			}
		}
	}()
	return nil
}
