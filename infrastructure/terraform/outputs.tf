output "eks_cluster_name" {
  value = module.eks.cluster_name
}
output "database_writer_endpoint" {
  value = aws_rds_cluster.postgres.endpoint
}
output "redis_primary_endpoint" {
  value = aws_elasticache_replication_group.redis.primary_endpoint_address
}
output "documents_bucket" {
  value = aws_s3_bucket.documents.id
}
output "jobs_queue_url" {
  value = aws_sqs_queue.jobs.url
}
output "jobs_dlq_url" {
  value = aws_sqs_queue.jobs_dlq.url
}
output "application_secret_arn" {
  value = aws_secretsmanager_secret.application.arn
}
output "waf_acl_arn" {
  value = aws_wafv2_web_acl.edge.arn
  description = "Associate with the AWS Load Balancer Controller ALB after ingress creation."
}
