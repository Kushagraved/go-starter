// Package sqs keeps every queue SDK detail — option structs, receipt handles,
// attribute names — out of application code. Swapping brokers means rewriting
// this package only.
package sqs

import (
	"context"
	"strconv"

	"go-starter/internal/infrastructure"
	"go-starter/internal/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/pkg/errors"
)

// NewConsumerClient resolves the URL for queueName and returns a consumer builder.
// autoCommit=false defers deletion until the message is processed successfully.
func NewConsumerClient(ctx context.Context, sqsClient *sqs.Client, queueName string, autoCommit bool, loggerObj logger.Logger) (infrastructure.ListenToEvents, error) {
	if sqsClient == nil {
		return nil, errors.New("sqs client is nil")
	}

	queueURL, err := sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(queueName)})
	if err != nil {
		return nil, errors.Wrapf(err, "error resolving queue url for %s", queueName)
	}

	loggerObj.Info(ctx, "SQS", "NewConsumerClient", "resolved queue url",
		logger.NewString("queue_name", queueName), logger.NewString("queue_url", *queueURL.QueueUrl))

	return &consumerClient{
		queueName:  queueName,
		queueUrl:   *queueURL.QueueUrl,
		sqs:        sqsClient,
		autoCommit: autoCommit,
		logger:     loggerObj,
	}, nil
}

func (c *consumerClient) WithMaxNumberOfMessages(maxMessages int32) infrastructure.ListenToEvents {
	c.maxMessages = maxMessages
	return c
}

func (c *consumerClient) WithVisibilityTimeout(visibilityTimeout int32) infrastructure.ListenToEvents {
	c.visibilityTimeout = visibilityTimeout
	return c
}

func (c *consumerClient) WithWaitTimeSeconds(waitTime int32) infrastructure.ListenToEvents {
	c.waitTime = waitTime
	return c
}

func (c *consumerClient) WithExponentialBackoff() infrastructure.ListenToEvents {
	c.requireExponentialBackoff = true
	return c
}

// poll fetches one batch of messages, long-polling when WaitTimeSeconds is set.
func (c *consumerClient) poll(ctx context.Context) ([]infrastructure.EventResponse, error) {
	output, err := c.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(c.queueUrl),
		// ApproximateReceiveCount drives the retry count and the backoff.
		MessageSystemAttributeNames: []types.MessageSystemAttributeName{
			types.MessageSystemAttributeNameApproximateReceiveCount,
		},
		MaxNumberOfMessages: c.maxMessages,
		VisibilityTimeout:   c.visibilityTimeout,
		WaitTimeSeconds:     c.waitTime,
	})
	if err != nil {
		return nil, err
	}

	events := make([]infrastructure.EventResponse, 0, len(output.Messages))
	for _, message := range output.Messages {
		retryCount, _ := strconv.ParseInt(
			message.Attributes[string(types.MessageSystemAttributeNameApproximateReceiveCount)], 10, 64)

		var body []byte
		if message.Body != nil {
			body = []byte(*message.Body)
		}

		events = append(events, infrastructure.EventResponse{
			RawData:        body,
			QueueName:      c.queueName,
			CommitMetadata: message.ReceiptHandle,
			RetryCount:     retryCount,
		})
	}
	return events, nil
}

// commit deletes a processed message from the queue.
func (c *consumerClient) commit(ctx context.Context, job processMetadata) error {
	receiptHandle, ok := job.CommitMetadata.(*string)
	if !ok || receiptHandle == nil {
		return errors.Errorf("invalid receipt handle for queue %s", c.queueName)
	}

	_, err := c.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueUrl),
		ReceiptHandle: receiptHandle,
	})
	return errors.Wrapf(err, "error deleting message from queue %s", c.queueName)
}

// NewProducerClient wraps an SQS client for publishing.
func NewProducerClient(sqsClient *sqs.Client, loggerObj logger.Logger) infrastructure.PushEvents {
	return &producerClient{sqs: sqsClient, logger: loggerObj}
}

func (p *producerClient) Produce(ctx context.Context, event *infrastructure.Message) error {
	if event == nil {
		return errors.New("event is nil")
	}

	result, err := p.sqs.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(event.Topic)})
	if err != nil {
		return errors.Wrapf(err, "error resolving queue url for %s", event.Topic)
	}

	message := &sqs.SendMessageInput{
		QueueUrl:    result.QueueUrl,
		MessageBody: aws.String(string(event.Data)),
	}
	if event.DeliveryDelay.Seconds() > 0 {
		message.DelaySeconds = int32(event.DeliveryDelay.Seconds())
	}

	if _, err = p.sqs.SendMessage(ctx, message); err != nil {
		return errors.Wrapf(err, "error publishing to %s", event.Topic)
	}
	return nil
}
