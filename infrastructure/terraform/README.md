# AWS foundation

This root provisions the production-shaped VPC, EKS application/GPU node groups, Aurora PostgreSQL, encrypted Redis, S3, SQS/DLQ, KMS, Secrets Manager placeholder, optional SES identity, and a regional WAF policy.

No secret values are managed here. Populate the created Secrets Manager secret through an approved delivery system, install the AWS Load Balancer Controller and Karpenter, and associate the output WAF ACL with the resulting ALB. Use a remote encrypted Terraform backend and reviewed `*.tfvars` in each environment; never commit a plan file or state.
