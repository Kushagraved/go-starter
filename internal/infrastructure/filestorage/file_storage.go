// Package filestorage implements infrastructure.FileStorage on top of S3.
package filestorage

import (
	"bytes"
	"context"

	"go-starter/internal/constants"
	"go-starter/internal/infrastructure"
	"go-starter/internal/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
)

type client struct {
	s3     *s3.Client
	logger logger.Logger
}

// New builds the S3-backed FileStorage.
func New(s3Client *s3.Client, loggerObj logger.Logger) infrastructure.FileStorage {
	return client{s3: s3Client, logger: loggerObj}
}

func (c client) StoreFile(ctx context.Context, bucket, key string, data []byte, params *infrastructure.FileStorageParams) (string, error) {
	c.logger.Info(ctx, "FileStorage", "StoreFile", "storing object",
		logger.NewString("bucket", bucket), logger.NewString("key", key), logger.NewInt("size", len(data)))

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}
	if params != nil && params.ContentType != "" {
		input.ContentType = aws.String(params.ContentType)
	}

	if _, err := c.s3.PutObject(ctx, input); err != nil {
		return "", errors.Wrap(err, "error uploading object")
	}
	return key, nil
}

func (c client) GetFile(ctx context.Context, bucket, key string) ([]byte, error) {
	res, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, constants.RecordDoesntExists
		}
		return nil, errors.Wrap(err, "error downloading object")
	}
	defer res.Body.Close()

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(res.Body); err != nil {
		return nil, errors.Wrap(err, "error reading object body")
	}
	return buf.Bytes(), nil
}

func (c client) RemoveFile(ctx context.Context, bucket, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c client) ListObjects(ctx context.Context, bucket, prefix string) ([]infrastructure.ListObjectsResponse, error) {
	var objects []infrastructure.ListObjectsResponse

	paginator := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error listing objects")
		}
		for _, object := range page.Contents {
			if object.Key == nil || object.Size == nil || *object.Size == 0 {
				continue
			}
			entry := infrastructure.ListObjectsResponse{FileName: *object.Key}
			if object.LastModified != nil {
				entry.LastModified = *object.LastModified
			}
			objects = append(objects, entry)
		}
	}
	return objects, nil
}

// isNotFound reports whether err means "the object is not there".
func isNotFound(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return true
		}
	}
	return false
}
