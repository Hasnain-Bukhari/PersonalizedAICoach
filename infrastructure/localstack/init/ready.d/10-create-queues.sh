#!/bin/sh
set -eu

awslocal sqs create-queue --queue-name coach-agent-jobs-dlq >/dev/null
DLQ_URL="$(awslocal sqs get-queue-url --queue-name coach-agent-jobs-dlq --query QueueUrl --output text)"
DLQ_ARN="$(awslocal sqs get-queue-attributes --queue-url "${DLQ_URL}" --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"
awslocal sqs create-queue \
  --queue-name coach-agent-jobs \
  --attributes "{\"VisibilityTimeout\":\"120\",\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"${DLQ_ARN}\\\",\\\"maxReceiveCount\\\":\\\"5\\\"}\"}" >/dev/null
