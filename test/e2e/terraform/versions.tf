terraform {
  required_version = ">= 1.5.7"

  backend "s3" {
    bucket = "ecr-secret-operator-e2e-tfstate"
    region = "us-east-2"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    rhcs = {
      source  = "terraform-redhat/rhcs"
      version = ">= 1.7.7"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.5"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      "app-code"      = "MOBB-001"
      "service-phase" = "lab"
      "owner"         = "github-actions"
      "repo"          = "ecr-secret-operator"
      "pr"            = var.pr_number
    }
  }
}

provider "rhcs" {}
