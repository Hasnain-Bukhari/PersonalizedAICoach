variable "project" {
  type = string
  default = "ai-learning-coach"
}
variable "environment" {
  type = string
  default = "production"
}
variable "aws_region" {
  type = string
  default = "us-east-1"
}
variable "vpc_cidr" {
  type = string
  default = "10.40.0.0/16"
}
variable "cluster_version" {
  type = string
  default = "1.32"
}
variable "database_name" {
  type = string
  default = "coach"
}
variable "database_master_username" {
  type = string
  default = "coach_admin"
}
variable "aurora_min_acu" {
  type = number
  default = 2
}
variable "aurora_max_acu" {
  type = number
  default = 32
}
variable "api_node_instance_types" {
  type = list(string)
  default = ["m7i.large"]
}
variable "gpu_node_instance_types" {
  type = list(string)
  default = ["g6.xlarge"]
}
variable "ses_domain" {
  type = string
  default = ""
  description = "Optional verified email domain. DNS records must be created externally."
}
