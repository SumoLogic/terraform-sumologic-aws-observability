# AWS Observability Terraform Module

This Terraform module deploys the [Sumo Logic AWS Observability Solution](https://help.sumologic.com/docs/observability/aws/) — a full-stack observability solution for AWS environments. It configures AWS collection infrastructure and installs Sumo Logic apps, monitors, dashboards, and field extraction rules for the following AWS services and supporting apps:

- Application Load Balancer (ALB)
- Classic Load Balancer (ELB)
- Network Load Balancer (NLB)
- API Gateway
- CloudTrail
- DynamoDB
- EC2
- ECS (Without Container Insights and Traces)
- ECS (With Container Insights and Traces)
- ElastiCache
- Amazon Overview
- Lambda
- RDS
- SNS
- SQS
- Host Metrics (EC2)

## Usage

### Configuration (`main.auto.tfvars`)

Create a `main.auto.tfvars` file with your environment-specific values:

```hcl
sumologic_environment     = "us2"                  # Sumo Logic deployment: au, ca, ch, de, eu, fed, jp, kr, us1, us2
sumologic_access_id       = "<YOUR SUMO ACCESS ID>"
sumologic_access_key      = "<YOUR SUMO ACCESS KEY>"
sumologic_organization_id = "<YOUR SUMO ORG ID>"
aws_account_alias         = "<AWS_ACCOUNT_ALIAS>"           # Lowercase letters and numbers only
```

```hcl
provider "sumologic" {
  environment = var.sumologic_environment
  access_id   = var.sumologic_access_id
  access_key  = var.sumologic_access_key
}

provider "aws" {
  region = "us-east-1"
}

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



For multi-account or multi-region deployments, use the submodules directly:

```hcl
# Install apps once per Sumo Logic org
module "apps" {
  source  = "SumoLogic/aws-observability/sumologic//modules/apps"
  version = ">= 1.0.0"
  ...
}

# Install collection once per AWS account/region
module "collection" {
  source  = "SumoLogic/aws-observability/sumologic//modules/collection"
  version = ">= 1.0.0"
  ...
}
```

See the [`examples/`](./examples) directory for complete working configurations.

## Submodules

| Name | Description |
|------|-------------|
| [modules/apps](./modules/apps) | Installs Sumo Logic apps, monitors, metric rules, FERs, and the AWS Observability hierarchy. Deploy once per Sumo Logic organization. |
| [modules/collection](modules/collections) | Creates AWS collection infrastructure (CloudTrail, ELB, CloudWatch, Kinesis Firehose sources) and Sumo Logic collector. Deploy once per AWS account/region. |

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.5.7 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.16.2, < 7.0.0 |
| <a name="requirement_random"></a> [random](#requirement\_random) | >= 3.1.0 |
| <a name="requirement_sumologic"></a> [sumologic](#requirement\_sumologic) | >= 3.3.0, < 4.0.0 |
| <a name="requirement_time"></a> [time](#requirement\_time) | >= 0.11.1 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_app-module"></a> [app-module](#module\_app-module) | ./modules/apps | n/a |
| <a name="module_collection-module"></a> [collection-module](#module\_collection-module) | ./modules/collections | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_aws_account_alias"></a> [aws\_account\_alias](#input\_aws\_account\_alias) | Provide the Name/Alias for the AWS environment from which you are collecting data. This name will appear in the Sumo Logic Explorer View, metrics, and logs.<br/>            If you are going to deploy the solution in multiple AWS accounts then this value has to be overidden at main.tf file.<br/>            Do not include special characters in the alias. | `string` | n/a | yes |
| <a name="input_aws_resource_tags"></a> [aws\_resource\_tags](#input\_aws\_resource\_tags) | Map of tags to apply to all AWS resources provisioned through the AWS Observability Solution | `map(string)` | `{}` | no |
| <a name="input_classic_lb_log_source_url"></a> [classic\_lb\_log\_source\_url](#input\_classic\_lb\_log\_source\_url) | Required if you are already collecting Classic LB logs. Provide the existing Sumo Logic Classic LB Source API URL. | `string` | `""` | no |
| <a name="input_cloudtrail_source_url"></a> [cloudtrail\_source\_url](#input\_cloudtrail\_source\_url) | Required if you are already collecting CloudTrail logs. Provide the existing Sumo Logic CloudTrail Source API URL. | `string` | `""` | no |
| <a name="input_cloudwatch_logs_source_url"></a> [cloudwatch\_logs\_source\_url](#input\_cloudwatch\_logs\_source\_url) | Required if you are already collecting CloudWatch Logs. Provide the existing Sumo Logic Logs Source API URL. e.g. https://api.us2.sumologic.com/api/v1/collectors/<collectorId>/sources/<sourceId> | `string` | `""` | no |
| <a name="input_cloudwatch_metrics_source_url"></a> [cloudwatch\_metrics\_source\_url](#input\_cloudwatch\_metrics\_source\_url) | Required if you are already collecting CloudWatch Metrics. Provide the existing Sumo Logic Metrics Source API URL. e.g. https://api.us2.sumologic.com/api/v1/collectors/<collectorId>/sources/<sourceId> | `string` | `""` | no |
| <a name="input_elb_log_source_url"></a> [elb\_log\_source\_url](#input\_elb\_log\_source\_url) | Required if you are already collecting ALB logs. Provide the existing Sumo Logic ALB Source API URL. | `string` | `""` | no |
| <a name="input_sumologic_access_id"></a> [sumologic\_access\_id](#input\_sumologic\_access\_id) | Sumo Logic Access ID. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key | `string` | n/a | yes |
| <a name="input_sumologic_access_key"></a> [sumologic\_access\_key](#input\_sumologic\_access\_key) | Sumo Logic Access Key. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key | `string` | n/a | yes |
| <a name="input_sumologic_environment"></a> [sumologic\_environment](#input\_sumologic\_environment) | Enter au, ca, ch, de, eu, esc, fed, jp, kr, us1 or us2. For more information on Sumo Logic deployments visit https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security | `string` | n/a | yes |
| <a name="input_sumologic_environment_base_url"></a> [sumologic\_environment\_base\_url](#input\_sumologic\_environment\_base\_url) | Base URL for custom Sumo Logic environments (e.g., 'https://api.ch.sumologic.com/api/' for Switzerland). If provided, this takes precedence over the sumologic\_environment parameter. Leave empty for standard deployments. | `string` | `null` | no |
| <a name="input_sumologic_organization_id"></a> [sumologic\_organization\_id](#input\_sumologic\_organization\_id) | You can find your org on the Preferences page in the Sumo Logic UI. For more information, see the Preferences Page topic. Your org ID will be used to configure the IAM Role for Sumo Logic AWS Sources."<br/>            For more details, visit https://help.sumologic.com/01Start-Here/05Customize-Your-Sumo-Logic-Experience/Preferences-Page | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_Apps"></a> [Apps](#output\_Apps) | All outputs related to apps. |
| <a name="output_Collection"></a> [Collection](#output\_Collection) | All outputs related to collection and sources. |
<!-- END_TF_DOCS -->