## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.5.7 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.16.2, < 7.0.0 |
| <a name="requirement_random"></a> [random](#requirement\_random) | >= 3.1.0 |
| <a name="requirement_sumologic"></a> [sumologic](#requirement\_sumologic) | >= 3.3.0, < 4.0.0 |
| <a name="requirement_time"></a> [time](#requirement\_time) | >= 0.11.1 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_sumologic"></a> [sumologic](#provider\_sumologic) | >= 3.3.0, < 4.0.0 |
| <a name="provider_time"></a> [time](#provider\_time) | >= 0.11.1 |

## Modules

| Name | Source | Version |
| ---- | ------ | ------- |
| <a name="module_sumo-module"></a> [sumo-module](#module\_sumo-module) | ../../modules/apps | n/a |

## Resources

| Name | Type |
| ---- | ---- |
| [sumologic_field.account](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.accountid](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.apiid](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.apiname](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.cacheclusterid](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.clustername](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.dbclusteridentifier](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.dbidentifier](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.dbinstanceidentifier](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.functionname](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.instanceid](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.loadbalancer](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.loadbalancername](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.namespace](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.networkloadbalancer](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.queuename](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.region](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.tablename](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field.topicname](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityALBCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityAlbAccessLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityApiGatewayAccessLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityApiGatewayCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityCLBCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityDynamoDBCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityEC2CloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityECSCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityElastiCacheCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityElbAccessLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityFieldExtractionRule](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityGenericCloudWatchLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityLambdaCloudWatchLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityNLBCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilityRdsCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilitySNSCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [sumologic_field_extraction_rule.AwsObservabilitySQSCloudTrailLogsFER](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/field_extraction_rule) | resource |
| [time_sleep.wait_for_10_seconds](https://registry.terraform.io/providers/hashicorp/time/latest/docs/resources/sleep) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_sumo_api_endpoint"></a> [sumo\_api\_endpoint](#input\_sumo\_api\_endpoint) | Sumo Logic API endpoint URL. E.g. https://api.us2.sumologic.com/api/ | `string` | `""` | no |
| <a name="input_sumologic_access_id"></a> [sumologic\_access\_id](#input\_sumologic\_access\_id) | Sumo Logic Access ID. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key | `string` | n/a | yes |
| <a name="input_sumologic_access_key"></a> [sumologic\_access\_key](#input\_sumologic\_access\_key) | Sumo Logic Access Key. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key | `string` | n/a | yes |
| <a name="input_sumologic_environment"></a> [sumologic\_environment](#input\_sumologic\_environment) | Enter au, ca, ch, de, eu, esc, fed, jp, kr, us1 or us2. For more information on Sumo Logic deployments visit https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_hierarchy_id"></a> [hierarchy\_id](#output\_hierarchy\_id) | ID of the AWS Observability Entity Inspector hierarchy. |
| <a name="output_installed_apps"></a> [installed\_apps](#output\_installed\_apps) | Map of apps installed into the Sumo Logic Installed Apps catalog. |
| <a name="output_sumologic_field_account"></a> [sumologic\_field\_account](#output\_sumologic\_field\_account) | Sumo Logic Account field ID. |
| <a name="output_sumologic_field_accountid"></a> [sumologic\_field\_accountid](#output\_sumologic\_field\_accountid) | Sumo Logic accountid field ID. |
| <a name="output_sumologic_field_apiid"></a> [sumologic\_field\_apiid](#output\_sumologic\_field\_apiid) | Sumo Logic apiid field ID. |
| <a name="output_sumologic_field_apiname"></a> [sumologic\_field\_apiname](#output\_sumologic\_field\_apiname) | Sumo Logic apiname field ID. |
| <a name="output_sumologic_field_cacheclusterid"></a> [sumologic\_field\_cacheclusterid](#output\_sumologic\_field\_cacheclusterid) | Sumo Logic cacheclusterid field ID. |
| <a name="output_sumologic_field_clustername"></a> [sumologic\_field\_clustername](#output\_sumologic\_field\_clustername) | Sumo Logic clustername field ID. |
| <a name="output_sumologic_field_dbclusteridentifier"></a> [sumologic\_field\_dbclusteridentifier](#output\_sumologic\_field\_dbclusteridentifier) | Sumo Logic dbclusteridentifier field ID. |
| <a name="output_sumologic_field_dbidentifier"></a> [sumologic\_field\_dbidentifier](#output\_sumologic\_field\_dbidentifier) | Sumo Logic dbidentifier field ID. |
| <a name="output_sumologic_field_dbinstanceidentifier"></a> [sumologic\_field\_dbinstanceidentifier](#output\_sumologic\_field\_dbinstanceidentifier) | Sumo Logic dbinstanceidentifier field ID. |
| <a name="output_sumologic_field_extraction_rule_alb"></a> [sumologic\_field\_extraction\_rule\_alb](#output\_sumologic\_field\_extraction\_rule\_alb) | ALB Access Logs FER ID. |
| <a name="output_sumologic_field_extraction_rule_alb_cloudtrail"></a> [sumologic\_field\_extraction\_rule\_alb\_cloudtrail](#output\_sumologic\_field\_extraction\_rule\_alb\_cloudtrail) | ALB CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_apigateway"></a> [sumologic\_field\_extraction\_rule\_apigateway](#output\_sumologic\_field\_extraction\_rule\_apigateway) | API Gateway CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_apigateway_access_logs"></a> [sumologic\_field\_extraction\_rule\_apigateway\_access\_logs](#output\_sumologic\_field\_extraction\_rule\_apigateway\_access\_logs) | API Gateway Access Logs FER ID. |
| <a name="output_sumologic_field_extraction_rule_clb_cloudtrail"></a> [sumologic\_field\_extraction\_rule\_clb\_cloudtrail](#output\_sumologic\_field\_extraction\_rule\_clb\_cloudtrail) | CLB CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_cw"></a> [sumologic\_field\_extraction\_rule\_cw](#output\_sumologic\_field\_extraction\_rule\_cw) | Generic CloudWatch FER ID. |
| <a name="output_sumologic_field_extraction_rule_dynamodb"></a> [sumologic\_field\_extraction\_rule\_dynamodb](#output\_sumologic\_field\_extraction\_rule\_dynamodb) | DynamoDB CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_ec2metrics"></a> [sumologic\_field\_extraction\_rule\_ec2metrics](#output\_sumologic\_field\_extraction\_rule\_ec2metrics) | EC2 CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_ecs"></a> [sumologic\_field\_extraction\_rule\_ecs](#output\_sumologic\_field\_extraction\_rule\_ecs) | ECS CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_elasticache"></a> [sumologic\_field\_extraction\_rule\_elasticache](#output\_sumologic\_field\_extraction\_rule\_elasticache) | ElastiCache CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_elb"></a> [sumologic\_field\_extraction\_rule\_elb](#output\_sumologic\_field\_extraction\_rule\_elb) | CLB Access Logs FER ID. |
| <a name="output_sumologic_field_extraction_rule_lambda"></a> [sumologic\_field\_extraction\_rule\_lambda](#output\_sumologic\_field\_extraction\_rule\_lambda) | Lambda CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_lambda_cw"></a> [sumologic\_field\_extraction\_rule\_lambda\_cw](#output\_sumologic\_field\_extraction\_rule\_lambda\_cw) | Lambda CloudWatch FER ID. |
| <a name="output_sumologic_field_extraction_rule_nlb_cloudtrail"></a> [sumologic\_field\_extraction\_rule\_nlb\_cloudtrail](#output\_sumologic\_field\_extraction\_rule\_nlb\_cloudtrail) | NLB CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_rds"></a> [sumologic\_field\_extraction\_rule\_rds](#output\_sumologic\_field\_extraction\_rule\_rds) | RDS CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_sns"></a> [sumologic\_field\_extraction\_rule\_sns](#output\_sumologic\_field\_extraction\_rule\_sns) | SNS CloudTrail FER ID. |
| <a name="output_sumologic_field_extraction_rule_sqs"></a> [sumologic\_field\_extraction\_rule\_sqs](#output\_sumologic\_field\_extraction\_rule\_sqs) | SQS CloudTrail FER ID. |
| <a name="output_sumologic_field_functionname"></a> [sumologic\_field\_functionname](#output\_sumologic\_field\_functionname) | Sumo Logic functionname field ID. |
| <a name="output_sumologic_field_instanceid"></a> [sumologic\_field\_instanceid](#output\_sumologic\_field\_instanceid) | Sumo Logic instanceid field ID. |
| <a name="output_sumologic_field_loadbalancer"></a> [sumologic\_field\_loadbalancer](#output\_sumologic\_field\_loadbalancer) | Sumo Logic loadbalancer field ID. |
| <a name="output_sumologic_field_loadbalancername"></a> [sumologic\_field\_loadbalancername](#output\_sumologic\_field\_loadbalancername) | Sumo Logic loadbalancername field ID. |
| <a name="output_sumologic_field_namespace"></a> [sumologic\_field\_namespace](#output\_sumologic\_field\_namespace) | Sumo Logic namespace field ID. |
| <a name="output_sumologic_field_networkloadbalancer"></a> [sumologic\_field\_networkloadbalancer](#output\_sumologic\_field\_networkloadbalancer) | Sumo Logic networkloadbalancer field ID. |
| <a name="output_sumologic_field_queuename"></a> [sumologic\_field\_queuename](#output\_sumologic\_field\_queuename) | Sumo Logic queuename field ID. |
| <a name="output_sumologic_field_region"></a> [sumologic\_field\_region](#output\_sumologic\_field\_region) | Sumo Logic Region field ID. |
| <a name="output_sumologic_field_tablename"></a> [sumologic\_field\_tablename](#output\_sumologic\_field\_tablename) | Sumo Logic tablename field ID. |
| <a name="output_sumologic_field_topicname"></a> [sumologic\_field\_topicname](#output\_sumologic\_field\_topicname) | Sumo Logic topicname field ID. |
