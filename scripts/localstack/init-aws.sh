#!/bin/bash
# LocalStack bootstrap.
#
# LocalStack runs every executable in /etc/localstack/init/ready.d once it is ready
# to serve requests, so this is where local buckets and queues are created.
# Keep it in sync with config/dev/config.yaml.
set -euo pipefail

REGION="${AWS_DEFAULT_REGION:-us-east-1}"
BUCKET="go-starter-local"

echo "[init-aws] creating s3 bucket: ${BUCKET}"
awslocal s3api create-bucket --bucket "${BUCKET}" --region "${REGION}"

# Add one line per queue, matching settings.sqs.<queue>.queue_name in config.
echo "[init-aws] creating sqs queue: test-events"
awslocal sqs create-queue --queue-name "test-events" --region "${REGION}" >/dev/null

echo "[init-aws] done"
