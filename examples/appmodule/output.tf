output "installed_apps" {
  value       = module.sumo-module.installed_apps
  description = "Map of apps installed into the Sumo Logic Installed Apps catalog."
}

output "hierarchy_id" {
  value       = module.sumo-module.hierarchy_id
  description = "ID of the AWS Observability Entity Inspector hierarchy."
}

# Field IDs
output "sumologic_field_account" {
  value       = sumologic_field.account.id
  description = "Sumo Logic Account field ID."
}

output "sumologic_field_region" {
  value       = sumologic_field.region.id
  description = "Sumo Logic Region field ID."
}

output "sumologic_field_accountid" {
  value       = sumologic_field.accountid.id
  description = "Sumo Logic accountid field ID."
}

output "sumologic_field_namespace" {
  value       = sumologic_field.namespace.id
  description = "Sumo Logic namespace field ID."
}

output "sumologic_field_loadbalancer" {
  value       = sumologic_field.loadbalancer.id
  description = "Sumo Logic loadbalancer field ID."
}

output "sumologic_field_loadbalancername" {
  value       = sumologic_field.loadbalancername.id
  description = "Sumo Logic loadbalancername field ID."
}

output "sumologic_field_apiname" {
  value       = sumologic_field.apiname.id
  description = "Sumo Logic apiname field ID."
}

output "sumologic_field_tablename" {
  value       = sumologic_field.tablename.id
  description = "Sumo Logic tablename field ID."
}

output "sumologic_field_instanceid" {
  value       = sumologic_field.instanceid.id
  description = "Sumo Logic instanceid field ID."
}

output "sumologic_field_clustername" {
  value       = sumologic_field.clustername.id
  description = "Sumo Logic clustername field ID."
}

output "sumologic_field_cacheclusterid" {
  value       = sumologic_field.cacheclusterid.id
  description = "Sumo Logic cacheclusterid field ID."
}

output "sumologic_field_functionname" {
  value       = sumologic_field.functionname.id
  description = "Sumo Logic functionname field ID."
}

output "sumologic_field_networkloadbalancer" {
  value       = sumologic_field.networkloadbalancer.id
  description = "Sumo Logic networkloadbalancer field ID."
}

output "sumologic_field_dbidentifier" {
  value       = sumologic_field.dbidentifier.id
  description = "Sumo Logic dbidentifier field ID."
}

output "sumologic_field_topicname" {
  value       = sumologic_field.topicname.id
  description = "Sumo Logic topicname field ID."
}

output "sumologic_field_queuename" {
  value       = sumologic_field.queuename.id
  description = "Sumo Logic queuename field ID."
}

output "sumologic_field_dbclusteridentifier" {
  value       = sumologic_field.dbclusteridentifier.id
  description = "Sumo Logic dbclusteridentifier field ID."
}

output "sumologic_field_dbinstanceidentifier" {
  value       = sumologic_field.dbinstanceidentifier.id
  description = "Sumo Logic dbinstanceidentifier field ID."
}

output "sumologic_field_apiid" {
  value       = sumologic_field.apiid.id
  description = "Sumo Logic apiid field ID."
}

# FER IDs
output "sumologic_field_extraction_rule_apigateway" {
  value       = sumologic_field_extraction_rule.AwsObservabilityApiGatewayCloudTrailLogsFER.id
  description = "API Gateway CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_apigateway_access_logs" {
  value       = sumologic_field_extraction_rule.AwsObservabilityApiGatewayAccessLogsFER.id
  description = "API Gateway Access Logs FER ID."
}

output "sumologic_field_extraction_rule_alb" {
  value       = sumologic_field_extraction_rule.AwsObservabilityAlbAccessLogsFER.id
  description = "ALB Access Logs FER ID."
}

output "sumologic_field_extraction_rule_alb_cloudtrail" {
  value       = sumologic_field_extraction_rule.AwsObservabilityALBCloudTrailLogsFER.id
  description = "ALB CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_elb" {
  value       = sumologic_field_extraction_rule.AwsObservabilityElbAccessLogsFER.id
  description = "CLB Access Logs FER ID."
}

output "sumologic_field_extraction_rule_clb_cloudtrail" {
  value       = sumologic_field_extraction_rule.AwsObservabilityCLBCloudTrailLogsFER.id
  description = "CLB CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_nlb_cloudtrail" {
  value       = sumologic_field_extraction_rule.AwsObservabilityNLBCloudTrailLogsFER.id
  description = "NLB CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_dynamodb" {
  value       = sumologic_field_extraction_rule.AwsObservabilityDynamoDBCloudTrailLogsFER.id
  description = "DynamoDB CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_elasticache" {
  value       = sumologic_field_extraction_rule.AwsObservabilityElastiCacheCloudTrailLogsFER.id
  description = "ElastiCache CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_ecs" {
  value       = sumologic_field_extraction_rule.AwsObservabilityECSCloudTrailLogsFER.id
  description = "ECS CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_ec2metrics" {
  value       = sumologic_field_extraction_rule.AwsObservabilityEC2CloudTrailLogsFER.id
  description = "EC2 CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_lambda" {
  value       = sumologic_field_extraction_rule.AwsObservabilityFieldExtractionRule.id
  description = "Lambda CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_lambda_cw" {
  value       = sumologic_field_extraction_rule.AwsObservabilityLambdaCloudWatchLogsFER.id
  description = "Lambda CloudWatch FER ID."
}

output "sumologic_field_extraction_rule_rds" {
  value       = sumologic_field_extraction_rule.AwsObservabilityRdsCloudTrailLogsFER.id
  description = "RDS CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_cw" {
  value       = sumologic_field_extraction_rule.AwsObservabilityGenericCloudWatchLogsFER.id
  description = "Generic CloudWatch FER ID."
}

output "sumologic_field_extraction_rule_sns" {
  value       = sumologic_field_extraction_rule.AwsObservabilitySNSCloudTrailLogsFER.id
  description = "SNS CloudTrail FER ID."
}

output "sumologic_field_extraction_rule_sqs" {
  value       = sumologic_field_extraction_rule.AwsObservabilitySQSCloudTrailLogsFER.id
  description = "SQS CloudTrail FER ID."
}
