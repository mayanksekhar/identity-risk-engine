variable "aws_region" {
  description = "AWS region to deploy into"
  type        = string
  default     = "us-east-1"
}

variable "project" {
  description = "Project name used for resource naming and tagging"
  type        = string
  default     = "ire"
}

variable "owner" {
  description = "Owner tag applied to all resources"
  type        = string
  default     = "mayanksekhar"
}

variable "eks_cluster_name" {
  description = "Name of the EKS cluster (must match eksctl cluster.yaml metadata.name)"
  type        = string
  default     = "ire-demo"
}

variable "container_image" {
  description = "Full image ref for the identity-risk-engine container (digest-pinned)"
  type        = string
  # Set at deploy time via:
  # terraform apply -var="container_image=ghcr.io/mayanksekhar/identity-risk-engine@sha256:<digest>"
  default     = "ghcr.io/mayanksekhar/identity-risk-engine:latest"
}

variable "eks_traffic_weight" {
  description = "Percentage of ALB traffic routed to EKS target group (remainder goes to ECS)"
  type        = number
  default     = 80
  validation {
    condition     = var.eks_traffic_weight >= 0 && var.eks_traffic_weight <= 100
    error_message = "eks_traffic_weight must be between 0 and 100."
  }
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "Availability zones to use for subnets"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}
