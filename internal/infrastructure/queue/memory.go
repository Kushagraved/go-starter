package queue

import (
	"context"
	"go-starter/internal/logger"
)

type memoryQueue struct {
	ch     chan Message
	logger logger.Logger
}

// NewMemoryQueue returns an in-process queue that implements both ListenToEvents and PushEvents.
// bufSize controls the channel buffer. Swap this for an SQS client in production.
func NewMemoryQueue(bufSize int, log logger.Logger) interface {
	ListenToEvents
	PushEvents
} {
	return &memoryQueue{
		ch:     make(chan Message, bufSize),
		logger: log,
	}
}

func (m *memoryQueue) Produce(_ context.Context, msg Message) error {
	m.ch <- msg
	return nil
}

// StartConsumer spins up workerCount goroutines that poll the channel,
// unmarshal each message, then call ProcessData — mirroring the SQS worker pool.
func (m *memoryQueue) StartConsumer(consumer Consumer, workerCount int) {
	jobs := make(chan processJob, workerCount)

	// fan-out: read from channel, unmarshal, push to jobs
	go func() {
		for msg := range m.ch {
			ctx := context.Background()
			data, err := consumer.Unmarshal(ctx, msg)
			if err != nil {
				m.logger.Error(ctx, "queue", consumer.GetTxnName(), "unmarshal error",
					logger.String("err", err.Error()))
				continue
			}
			jobs <- processJob{ctx: ctx, topic: msg.Topic, data: data}
		}
	}()

	// workers: process jobs concurrently
	for i := 0; i < workerCount; i++ {
		go func() {
			for j := range jobs {
				if err := consumer.ProcessData(j.ctx, j.topic, j.data); err != nil {
					m.logger.Error(j.ctx, "queue", consumer.GetTxnName(), "process error",
						logger.String("err", err.Error()))
				}
			}
		}()
	}
}

type processJob struct {
	ctx   context.Context
	topic string
	data  interface{}
}
