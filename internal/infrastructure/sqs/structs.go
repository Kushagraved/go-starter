package sqs

import (
	"go-starter/internal/infrastructure"
	"go-starter/internal/logger"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type consumerClient struct {
	queueName string
	queueUrl  string
	sqs       *sqs.Client
	logger    logger.Logger
	// autoCommit=false deletes the message only after ProcessData succeeds.
	autoCommit bool

	maxMessages               int32
	visibilityTimeout         int32
	waitTime                  int32
	requireExponentialBackoff bool
}

type producerClient struct {
	sqs    *sqs.Client
	logger logger.Logger
}

// processMetadata travels with a message through the internal pipeline.
type processMetadata struct {
	Topic          string
	RetryCount     int64
	CommitMetadata interface{}
	ProcessedData  interface{}
	processor      infrastructure.Consumer
}
