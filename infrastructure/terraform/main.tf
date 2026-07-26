data "aws_availability_zones" "available" { state = "available" }
data "aws_caller_identity" "current" {}

locals {
  name = "${var.project}-${var.environment}"
  azs  = slice(data.aws_availability_zones.available.names, 0, 3)
  tags = { Project = var.project, Environment = var.environment, ManagedBy = "terraform" }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.21.0"
  name    = local.name
  cidr    = var.vpc_cidr
  azs     = local.azs
  private_subnets  = [for i, _ in local.azs : cidrsubnet(var.vpc_cidr, 4, i)]
  database_subnets = [for i, _ in local.azs : cidrsubnet(var.vpc_cidr, 8, 64 + i)]
  public_subnets   = [for i, _ in local.azs : cidrsubnet(var.vpc_cidr, 8, 128 + i)]
  enable_nat_gateway = true
  single_nat_gateway = false
  enable_dns_hostnames = true
  enable_dns_support   = true
  public_subnet_tags = { "kubernetes.io/role/elb" = "1" }
  private_subnet_tags = { "kubernetes.io/role/internal-elb" = "1" }
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.36.0"
  cluster_name    = local.name
  cluster_version = var.cluster_version
  cluster_endpoint_public_access = true
  enable_cluster_creator_admin_permissions = true
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets
  cluster_addons = {
    coredns = { most_recent = true }
    eks-pod-identity-agent = { most_recent = true }
    kube-proxy = { most_recent = true }
    vpc-cni = {
      most_recent = true
      before_compute = true
    }
  }
  eks_managed_node_groups = {
    application = {
      ami_type       = "AL2023_x86_64_STANDARD"
      instance_types = var.api_node_instance_types
      min_size = 3
      max_size = 30
      desired_size = 3
      labels = { workload = "application" }
    }
    inference = {
      ami_type       = "AL2023_x86_64_NVIDIA"
      instance_types = var.gpu_node_instance_types
      min_size = 0
      max_size = 20
      desired_size = 0
      labels = { workload = "inference", accelerator = "nvidia" }
      taints = { gpu = { key = "nvidia.com/gpu", value = "true", effect = "NO_SCHEDULE" } }
    }
  }
}

resource "aws_security_group" "data" {
  name_prefix = "${local.name}-data-"
  description = "Data services reachable only from inside the VPC"
  vpc_id      = module.vpc.vpc_id
  ingress {
    description = "PostgreSQL"
    from_port = 5432
    to_port = 5432
    protocol = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }
  ingress {
    description = "Redis"
    from_port = 6379
    to_port = 6379
    protocol = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_rds_cluster" "postgres" {
  cluster_identifier = local.name
  engine = "aurora-postgresql"
  engine_mode = "provisioned"
  engine_version = "16.6"
  database_name = var.database_name
  master_username = var.database_master_username
  manage_master_user_password = true
  storage_encrypted = true
  backup_retention_period = 14
  preferred_backup_window = "03:00-04:00"
  copy_tags_to_snapshot = true
  deletion_protection = true
  skip_final_snapshot = false
  final_snapshot_identifier = "${local.name}-final"
  db_subnet_group_name = module.vpc.database_subnet_group_name
  vpc_security_group_ids = [aws_security_group.data.id]
  serverlessv2_scaling_configuration {
    min_capacity = var.aurora_min_acu
    max_capacity = var.aurora_max_acu
  }
}

resource "aws_rds_cluster_instance" "postgres" {
  count = 2
  identifier = "${local.name}-${count.index}"
  cluster_identifier = aws_rds_cluster.postgres.id
  instance_class = "db.serverless"
  engine = aws_rds_cluster.postgres.engine
  engine_version = aws_rds_cluster.postgres.engine_version
}

resource "aws_elasticache_subnet_group" "redis" {
  name = local.name
  subnet_ids = module.vpc.database_subnets
}
resource "aws_elasticache_replication_group" "redis" {
  replication_group_id = local.name
  description = "Learning Coach cache and ephemeral streams"
  engine = "redis"
  engine_version = "7.1"
  node_type = "cache.r7g.large"
  num_cache_clusters = 2
  automatic_failover_enabled = true
  multi_az_enabled = true
  transit_encryption_enabled = true
  at_rest_encryption_enabled = true
  subnet_group_name = aws_elasticache_subnet_group.redis.name
  security_group_ids = [aws_security_group.data.id]
}

resource "aws_s3_bucket" "documents" { bucket = "${local.name}-documents-${data.aws_caller_identity.current.account_id}" }
resource "aws_s3_bucket_versioning" "documents" {
  bucket = aws_s3_bucket.documents.id
  versioning_configuration {
    status = "Enabled"
  }
}
resource "aws_s3_bucket_server_side_encryption_configuration" "documents" {
  bucket = aws_s3_bucket.documents.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
      kms_master_key_id = aws_kms_key.application.arn
    }
    bucket_key_enabled = true
  }
}
resource "aws_s3_bucket_public_access_block" "documents" {
  bucket = aws_s3_bucket.documents.id
  block_public_acls = true
  block_public_policy = true
  ignore_public_acls = true
  restrict_public_buckets = true
}
resource "aws_s3_bucket_lifecycle_configuration" "documents" {
  bucket = aws_s3_bucket.documents.id
  rule {
    id = "abort-incomplete"
    status = "Enabled"
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

resource "aws_kms_key" "application" {
  description = "${local.name} application data"
  enable_key_rotation = true
  deletion_window_in_days = 30
}
resource "aws_kms_alias" "application" {
  name = "alias/${local.name}"
  target_key_id = aws_kms_key.application.key_id
}

resource "aws_sqs_queue" "jobs_dlq" {
  name = "${local.name}-jobs-dlq"
  kms_master_key_id = "alias/aws/sqs"
  message_retention_seconds = 1209600
}
resource "aws_sqs_queue" "jobs" {
  name = "${local.name}-jobs"
  kms_master_key_id = "alias/aws/sqs"
  visibility_timeout_seconds = 120
  receive_wait_time_seconds = 20
  redrive_policy = jsonencode({ deadLetterTargetArn = aws_sqs_queue.jobs_dlq.arn, maxReceiveCount = 5 })
}
resource "aws_sqs_queue_redrive_allow_policy" "jobs" {
  queue_url = aws_sqs_queue.jobs_dlq.id
  redrive_allow_policy = jsonencode({ redrivePermission = "byQueue", sourceQueueArns = [aws_sqs_queue.jobs.arn] })
}

resource "aws_secretsmanager_secret" "application" {
  name = "${local.name}/application"
  kms_key_id = aws_kms_key.application.arn
  recovery_window_in_days = 30
}
resource "aws_ses_domain_identity" "email" {
  count = var.ses_domain == "" ? 0 : 1
  domain = var.ses_domain
}

resource "aws_wafv2_web_acl" "edge" {
  name = local.name
  scope = "REGIONAL"
  default_action { allow {} }
  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name = local.name
    sampled_requests_enabled = true
  }
  rule {
    name = "AWSManagedRulesCommonRuleSet"
    priority = 10
    override_action { none {} }
    statement {
      managed_rule_group_statement {
        name = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name = "common"
      sampled_requests_enabled = true
    }
  }
  rule {
    name = "RateLimit"
    priority = 20
    action { block {} }
    statement {
      rate_based_statement {
        aggregate_key_type = "IP"
        limit = 2000
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name = "rate-limit"
      sampled_requests_enabled = true
    }
  }
}
