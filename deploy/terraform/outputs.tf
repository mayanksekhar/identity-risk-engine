output "vpc_id" {
  description = "VPC ID - used in eksctl cluster.yaml vpc.id"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "Public subnet IDs - used in eksctl cluster.yaml vpc.subnets.public"
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "Private subnet IDs - used in eksctl cluster.yaml vpc.subnets.private"
  value       = aws_subnet.private[*].id
}

output "eks_subnet_ids" {
  description = "All subnet IDs for EKS (public + private)"
  value       = concat(aws_subnet.public[*].id, aws_subnet.private[*].id)
}

output "private_subnet_1" {
  description = "First private subnet ID (us-east-1a) - used in eksctl cluster.yaml"
  value       = aws_subnet.private[0].id
}

output "private_subnet_2" {
  description = "Second private subnet ID (us-east-1b) - used in eksctl cluster.yaml"
  value       = aws_subnet.private[1].id
}

output "public_subnet_1" {
  description = "First public subnet ID (us-east-1a) - used in eksctl cluster.yaml"
  value       = aws_subnet.public[0].id
}

output "public_subnet_2" {
  description = "Second public subnet ID (us-east-1b) - used in eksctl cluster.yaml"
  value       = aws_subnet.public[1].id
}

output "alb_dns_name" {
  description = "ALB DNS name - the single entry point for all traffic"
  value       = aws_lb.main.dns_name
}

output "alb_arn" {
  description = "ALB ARN - needed by AWS Load Balancer Controller annotations"
  value       = aws_lb.main.arn
}

output "eks_target_group_arn" {
  description = "EKS target group ARN - referenced in K8s Service annotations"
  value       = aws_lb_target_group.eks.arn
}

output "ecs_target_group_arn" {
  description = "ECS target group ARN - registered by ECS service"
  value       = aws_lb_target_group.ecs.arn
}

output "ecs_cluster_name" {
  description = "ECS cluster name - used in CI deploy workflow"
  value       = aws_ecs_cluster.main.name
}

output "app_security_group_id" {
  description = "App security group ID - attach to EKS nodes via eksctl"
  value       = aws_security_group.app.id
}

output "traffic_split" {
  description = "Current ALB traffic split summary"
  value       = "EKS: ${var.eks_traffic_weight}% | ECS: ${100 - var.eks_traffic_weight}%"
}

output "deploy_instructions" {
  description = "Next steps after terraform apply"
  value       = <<-EOT
    1. Update deploy/eksctl/cluster.yaml with VPC/subnet IDs from this output
    2. Run: eksctl create cluster -f deploy/eksctl/cluster.yaml
    3. Install AWS Load Balancer Controller:
       helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
         -n kube-system \
         --set clusterName=${var.eks_cluster_name} \
         --set serviceAccount.create=false \
         --set serviceAccount.name=aws-load-balancer-controller
    4. Apply K8s manifests:
       kubectl apply -f deploy/k8s/namespace.yaml
       kubectl apply -f deploy/k8s/kyverno-policy.yaml -n production
       kubectl apply -f deploy/k8s/ -n staging
    5. ALB endpoint: http://${aws_lb.main.dns_name}
  EOT
}
