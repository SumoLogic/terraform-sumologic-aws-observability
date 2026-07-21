# terraform-sumologic-aws-observability

Terraform module for deploying the [Sumo Logic AWS Observability Solution](https://help.sumologic.com/docs/observability/aws/) — a full-stack observability solution for AWS environments covering ALB, ELB, NLB, API Gateway, CloudTrail, DynamoDB, EC2, ECS, ElastiCache, Lambda, RDS, SNS, and SQS.

## Usage

```hcl
module "aws_observability" {
  source  = "SumoLogic/aws-observability/sumologic"
  version = ">= 1.0.0"

  sumologic_environment     = "us2"
  sumologic_access_id       = var.sumologic_access_id
  sumologic_access_key      = var.sumologic_access_key
  sumologic_organization_id = var.sumologic_organization_id
  aws_account_alias         = "prod"
}
```

## Documentation

Full documentation, examples, and module reference are in the module source. See the [Sumo Logic AWS Observability docs](https://help.sumologic.com/docs/observability/aws/) for a complete guide.
