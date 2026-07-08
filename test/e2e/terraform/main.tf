data "aws_caller_identity" "current" {}

data "rhcs_versions" "latest" {
  search = "enabled='t' and rosa_enabled='t' and channel_group='stable' and hosted_control_plane_enabled='t'"
  order  = "id desc"
}

locals {
  openshift_version = data.rhcs_versions.latest.items[0].name
}

locals {
  common_tags = {
    "app-code"      = "MOBB-001"
    "service-phase" = "lab"
    "owner"         = "github-actions"
    "repo"          = "ecr-secret-operator"
    "pr"            = var.pr_number
  }
}

# --- VPC (single AZ to minimise cost) ---

module "vpc" {
  source  = "terraform-redhat/rosa-hcp/rhcs//modules/vpc"
  version = "1.7.4"

  name_prefix              = var.cluster_name
  availability_zones_count = 1
  vpc_cidr                 = "10.0.0.0/16"

  tags = merge(local.common_tags, {
    "e2e-test" = "true"
    "cluster"  = var.cluster_name
  })
}

# --- Cluster admin password ---

resource "random_password" "cluster_admin" {
  length           = 24
  special          = true
  override_special = "!#%&*()-_=+[]{}:?"
}

# --- VPC endpoint cleanup (runs between cluster and VPC destroy) ---

resource "null_resource" "vpc_endpoint_cleanup" {
  triggers = {
    vpc_id = module.vpc.vpc_id
    region = var.aws_region
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      echo "Cleaning up orphaned VPC endpoints and security groups..."

      ENDPOINTS=$(aws ec2 describe-vpc-endpoints \
        --filters "Name=vpc-id,Values=${self.triggers.vpc_id}" \
        --query 'VpcEndpoints[].VpcEndpointId' \
        --output text --region ${self.triggers.region} 2>/dev/null || true)

      if [ -n "$ENDPOINTS" ] && [ "$ENDPOINTS" != "None" ]; then
        echo "Deleting VPC endpoints: $ENDPOINTS"
        aws ec2 delete-vpc-endpoints \
          --vpc-endpoint-ids $ENDPOINTS \
          --region ${self.triggers.region}
        echo "Waiting for endpoints to be deleted..."
        sleep 30
      fi

      for sg in $(aws ec2 describe-security-groups \
        --filters "Name=vpc-id,Values=${self.triggers.vpc_id}" \
        --query 'SecurityGroups[?GroupName!=`default`].GroupId' \
        --output text --region ${self.triggers.region} 2>/dev/null); do
        echo "Deleting security group: $sg"
        aws ec2 delete-security-group --group-id "$sg" --region ${self.triggers.region} 2>/dev/null || true
      done

      echo "VPC endpoint cleanup complete."
    EOT
  }
}

# --- ROSA HCP cluster ---

module "rosa_hcp" {
  source  = "terraform-redhat/rosa-hcp/rhcs"
  version = "1.7.4"

  depends_on = [null_resource.vpc_endpoint_cleanup]

  cluster_name           = var.cluster_name
  openshift_version      = local.openshift_version
  machine_cidr           = module.vpc.cidr_block
  aws_subnet_ids         = concat(module.vpc.private_subnets, module.vpc.public_subnets)
  aws_availability_zones = module.vpc.availability_zones
  replicas               = var.replicas
  compute_machine_type   = var.compute_machine_type

  create_account_roles  = true
  account_role_prefix   = "${var.cluster_name}-acct"
  create_oidc           = true
  create_operator_roles = true
  operator_role_prefix  = "${var.cluster_name}-op"

  admin_credentials_username = "cluster-admin"
  admin_credentials_password = random_password.cluster_admin.result

  wait_for_create_complete = true

  tags = merge(local.common_tags, {
    "e2e-test" = "true"
  })
}

# --- IAM role for operator ECR access (OIDC federation) ---

data "aws_iam_openid_connect_provider" "rosa" {
  url = "https://${module.rosa_hcp.oidc_endpoint_url}"
}

resource "aws_iam_role" "operator_ecr" {
  name = "${var.cluster_name}-ecr-operator"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = data.aws_iam_openid_connect_provider.rosa.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${module.rosa_hcp.oidc_endpoint_url}:sub" = "system:serviceaccount:ecr-secret-operator:ecr-secret-operator-controller-manager"
        }
      }
    }]
  })

  tags = merge(local.common_tags, { "e2e-test" = "true" })
}

resource "aws_iam_role_policy" "operator_ecr" {
  name = "ecr-get-token"
  role = aws_iam_role.operator_ecr.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "ecr:GetAuthorizationToken"
      Resource = "*"
    }]
  })
}

# --- ECR repository ---

resource "aws_ecr_repository" "test" {
  name                 = var.ecr_repository_name
  image_tag_mutability = "MUTABLE"
  force_delete         = true

  tags = merge(local.common_tags, {
    "e2e-test" = "true"
    "cluster"  = var.cluster_name
  })
}
