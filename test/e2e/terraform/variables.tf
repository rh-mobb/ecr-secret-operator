variable "cluster_name" {
  description = "Name of the ROSA HCP cluster"
  type        = string
}

variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "us-east-2"
}

variable "compute_machine_type" {
  description = "EC2 instance type for worker nodes"
  type        = string
  default     = "m7i.xlarge"
}

variable "replicas" {
  description = "Number of worker nodes"
  type        = number
  default     = 2
}

variable "ecr_repository_name" {
  description = "Name of the ECR repository for testing"
  type        = string
  default     = "ecr-e2e-test-repo"
}

variable "pr_number" {
  description = "Pull request number (used for resource tagging)"
  type        = string
}
