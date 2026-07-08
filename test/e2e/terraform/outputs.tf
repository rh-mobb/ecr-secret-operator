output "cluster_name" {
  description = "ROSA HCP cluster name"
  value       = var.cluster_name
}

output "cluster_api_url" {
  description = "Cluster API server URL"
  value       = module.rosa_hcp.cluster_api_url
}

output "cluster_console_url" {
  description = "Cluster web console URL"
  value       = module.rosa_hcp.cluster_console_url
}

output "cluster_admin_password" {
  description = "Cluster admin password"
  value       = random_password.cluster_admin.result
  sensitive   = true
}

output "ecr_registry_url" {
  description = "ECR registry URL (account-level, without repo name)"
  value       = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com"
}

output "ecr_repository_url" {
  description = "Full ECR repository URL"
  value       = aws_ecr_repository.test.repository_url
}

output "aws_account_id" {
  description = "AWS account ID"
  value       = data.aws_caller_identity.current.account_id
}

output "operator_ecr_role_arn" {
  description = "IAM role ARN for the operator to assume via OIDC for ECR access"
  value       = aws_iam_role.operator_ecr.arn
}
