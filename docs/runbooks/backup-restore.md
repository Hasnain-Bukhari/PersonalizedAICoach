# Backup and restore runbook

Aurora automated backups retain 14 days and S3 versioning protects source documents. At least quarterly, restore Aurora into an isolated VPC, validate migrations and row counts, enable pgvector, and run tenant-isolation queries using the application role. Restore a representative private S3 object/version and prove it cannot be fetched anonymously.

Record recovery point, recovery time, checksums, and validation results. Never connect a restore exercise to production queues, email, identity callbacks, or model providers. Destroy the isolated environment through the approved Terraform workflow after evidence is retained.
