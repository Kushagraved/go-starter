// Package awsclient builds the AWS SDK clients.
//
// One aws.Config is loaded and shared by every service client. When
// `settings.aws.endpoint` is set, all clients point at that endpoint — that is how
// LocalStack is used locally without any code branching between environments.
// Leave it empty in real environments and the SDK resolves real AWS endpoints
// using the default credential chain.
package awsclient

import (
	"context"

	appconfig "go-starter/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pkg/errors"
)

// Clients holds the AWS service clients the app uses.
type Clients struct {
	S3  *s3.Client
	SQS *sqs.Client
}

// New loads the shared config and constructs the service clients.
func New(ctx context.Context, cfg appconfig.AWSCfg) (Clients, error) {
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}

	// Static credentials are for LocalStack. Leave them blank in real environments
	// so the default chain (IAM role, env, shared config) is used.
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return Clients{}, errors.Wrap(err, "failed to load aws config")
	}

	return Clients{
		S3: s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.Endpoint)
				// LocalStack serves buckets as a path segment, not a subdomain.
				o.UsePathStyle = true
			}
		}),
		SQS: sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.Endpoint)
			}
		}),
	}, nil
}
