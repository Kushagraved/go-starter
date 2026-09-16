package sqs

import (
	"context"

	"go-starter/internal/constants"
	"go-starter/internal/helpers/app"
	"go-starter/internal/infrastructure"
	"go-starter/internal/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gofrs/uuid"
)

/*
StartConsumer wires up the pipeline and returns immediately; it runs until ctx is
cancelled.

	poll ──▶ unmarshal ──▶ jobs ──▶ N workers ──▶ commit

Every stage recovers panics, so one malformed message can never take the process
down.
*/
func (c *consumerClient) StartConsumer(ctx context.Context, consumer infrastructure.Consumer, workerCount int) bool {
	if workerCount <= 0 {
		workerCount = 1
	}
	buffer := c.maxMessages
	if buffer <= 0 {
		buffer = 10
	}

	rawMessages := make(chan infrastructure.EventResponse, buffer)
	jobs := make(chan processMetadata, workerCount)

	go c.recoverable(ctx, "poll", func() { c.pollLoop(ctx, rawMessages) })
	go c.recoverable(ctx, "unmarshal", func() { c.unmarshalLoop(ctx, consumer, rawMessages, jobs) })
	for w := 0; w < workerCount; w++ {
		go c.recoverable(ctx, "worker", func() { c.worker(ctx, jobs) })
	}

	c.logger.Info(ctx, "SQS", "StartConsumer", "consumer started",
		logger.NewString("queue_name", c.queueName),
		logger.NewString("consumer", consumer.GetTxnName()),
		logger.NewInt("workers", workerCount))

	return true
}

// pollLoop long-polls the queue and forwards messages downstream.
func (c *consumerClient) pollLoop(ctx context.Context, out chan<- infrastructure.EventResponse) {
	for ctx.Err() == nil {
		messages, err := c.poll(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error(ctx, "SQS", "pollLoop", "error polling queue",
				logger.NewString("queue_name", c.queueName), logger.NewError(err))
			continue
		}

		for _, message := range messages {
			select {
			case <-ctx.Done():
				return
			case out <- message:
			}
		}
	}
}

// unmarshalLoop decodes each message and hands it to the worker pool.
func (c *consumerClient) unmarshalLoop(ctx context.Context, consumer infrastructure.Consumer,
	in <-chan infrastructure.EventResponse, jobs chan<- processMetadata) {

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-in:
			if len(message.RawData) == 0 {
				continue
			}

			msgCtx := c.withTraceID(ctx)
			data, err := consumer.Unmarshal(msgCtx, infrastructure.QueueMessageMetadata{
				Data:           message.RawData,
				Topic:          message.QueueName,
				CommitMetadata: message.CommitMetadata,
			}, c.logger)
			if err != nil {
				c.logger.Error(msgCtx, "SQS", "unmarshalLoop", "error un-marshalling message",
					logger.NewString("queue_name", c.queueName), logger.NewError(err))
				continue
			}

			select {
			case <-ctx.Done():
				return
			case jobs <- processMetadata{
				Topic:          message.QueueName,
				CommitMetadata: message.CommitMetadata,
				processor:      consumer,
				ProcessedData:  data,
				RetryCount:     message.RetryCount,
			}:
			}
		}
	}
}

func (c *consumerClient) worker(ctx context.Context, jobs <-chan processMetadata) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-jobs:
			c.process(ctx, job)
		}
	}
}

/*
process runs the consumer's logic for one message, then decides its fate:

  - success and autoCommit=false → delete the message;
  - failure with exponential backoff → extend the visibility timeout
    (timeout × 2 × receiveCount) so retries spread out instead of hot-looping;
  - failure without backoff → leave it to reappear after the normal timeout.
*/
func (c *consumerClient) process(parentCtx context.Context, job processMetadata) {
	ctx := c.withTraceID(parentCtx)

	defer func() {
		if r := recover(); r != nil {
			c.logger.Error(ctx, "SQS", "process", "recovered from panic",
				logger.NewInterface("panic", r), logger.NewString("trace", app.StackTrace()))
		}
	}()

	if err := job.processor.ProcessData(ctx, c.logger, job.Topic, job.ProcessedData, job.RetryCount); err != nil {
		c.logger.Error(ctx, "SQS", "process", "error processing message",
			logger.NewString("queue_name", c.queueName), logger.NewError(err))
		if c.requireExponentialBackoff && c.visibilityTimeout > 0 {
			c.backoff(ctx, job)
		}
		return
	}

	if !c.autoCommit {
		if err := c.commit(ctx, job); err != nil {
			c.logger.Error(ctx, "SQS", "process", "error committing message",
				logger.NewString("queue_name", c.queueName), logger.NewError(err))
		}
	}
}

// backoff pushes the message's next delivery further out on each retry.
func (c *consumerClient) backoff(ctx context.Context, job processMetadata) {
	receiptHandle, ok := job.CommitMetadata.(*string)
	if !ok || receiptHandle == nil {
		return
	}

	retryCount := job.RetryCount
	if retryCount < 1 {
		retryCount = 1
	}

	if _, err := c.sqs.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(c.queueUrl),
		ReceiptHandle:     receiptHandle,
		VisibilityTimeout: c.visibilityTimeout * 2 * int32(retryCount),
	}); err != nil {
		c.logger.Error(ctx, "SQS", "backoff", "error changing message visibility",
			logger.NewString("queue_name", c.queueName), logger.NewError(err))
	}
}

// withTraceID attaches a fresh trace ID so all logs for one message correlate.
func (c *consumerClient) withTraceID(parent context.Context) context.Context {
	traceID, err := uuid.NewV4()
	if err != nil {
		return parent
	}
	return context.WithValue(parent, constants.TraceID, traceID.String())
}

// recoverable runs fn, logging and swallowing any panic.
func (c *consumerClient) recoverable(ctx context.Context, stage string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error(ctx, "SQS", stage, "recovered from panic",
				logger.NewInterface("panic", r), logger.NewString("trace", app.StackTrace()))
		}
	}()
	fn()
}
