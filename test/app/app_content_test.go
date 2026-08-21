package app_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestContent_AppsInstalledInCatalog deploys and verifies all apps are present
// in the installed_apps output, confirming they were added to the Sumo Logic
// "Installed Apps" catalog (not a personal/admin folder).
func TestContent_AppsInstalledInCatalog(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "validate_apps", func() {
		validateInstalledApps(t, workingDir, expectedAppNames)
	})
}

// TestContent_HierarchyStructure deploys and validates that the AWS Observability
// Entity Inspector hierarchy is created in Sumo Logic.
func TestContent_HierarchyStructure(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "validate_hierarchy", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		validateHierarchyExists(t, opts)
	})
}

// TestContent_FERsExist deploys and validates that all 17 Field Extraction Rules
// were created with non-empty IDs.
func TestContent_FERsExist(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "validate_fers", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)

		fers := []struct {
			output string
			label  string
		}{
			{"sumologic_field_extraction_rule_apigateway", "APIGateway CloudTrail"},
			{"sumologic_field_extraction_rule_apigateway_access_logs", "APIGateway AccessLogs"},
			{"sumologic_field_extraction_rule_alb", "ALB AccessLogs"},
			{"sumologic_field_extraction_rule_alb_cloudtrail", "ALB CloudTrail"},
			{"sumologic_field_extraction_rule_elb", "CLB AccessLogs"},
			{"sumologic_field_extraction_rule_clb_cloudtrail", "CLB CloudTrail"},
			{"sumologic_field_extraction_rule_nlb_cloudtrail", "NLB CloudTrail"},
			{"sumologic_field_extraction_rule_dynamodb", "DynamoDB CloudTrail"},
			{"sumologic_field_extraction_rule_elasticache", "ElastiCache CloudTrail"},
			{"sumologic_field_extraction_rule_ecs", "ECS CloudTrail"},
			{"sumologic_field_extraction_rule_ec2metrics", "EC2 CloudTrail"},
			{"sumologic_field_extraction_rule_lambda", "Lambda CloudTrail"},
			{"sumologic_field_extraction_rule_lambda_cw", "Lambda CloudWatch"},
			{"sumologic_field_extraction_rule_rds", "RDS CloudTrail"},
			{"sumologic_field_extraction_rule_cw", "Generic CloudWatch"},
			{"sumologic_field_extraction_rule_sns", "SNS CloudTrail"},
			{"sumologic_field_extraction_rule_sqs", "SQS CloudTrail"},
		}

		passed, failed := 0, 0
		for _, f := range fers {
			id := terraform.Output(t, opts, f.output)
			if id == "" {
				t.Errorf("[fer] MISSING: %s (%s)", f.label, f.output)
				failed++
			} else {
				t.Logf("[fer] OK: %s id=%s", f.label, id)
				passed++
			}
		}
		t.Logf("[fer] %d/%d FERs validated", passed, passed+failed)
	})
}

// TestContent_FieldsExist deploys and validates that all 19 Sumo Logic fields
// were created or imported with non-empty IDs.
func TestContent_FieldsExist(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "validate_fields", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)

		fields := []struct {
			output    string
			fieldName string
		}{
			{"sumologic_field_account", "account"},
			{"sumologic_field_region", "region"},
			{"sumologic_field_accountid", "accountid"},
			{"sumologic_field_namespace", "namespace"},
			{"sumologic_field_loadbalancer", "loadbalancer"},
			{"sumologic_field_loadbalancername", "loadbalancername"},
			{"sumologic_field_apiname", "apiname"},
			{"sumologic_field_tablename", "tablename"},
			{"sumologic_field_instanceid", "instanceid"},
			{"sumologic_field_clustername", "clustername"},
			{"sumologic_field_cacheclusterid", "cacheclusterid"},
			{"sumologic_field_functionname", "functionname"},
			{"sumologic_field_networkloadbalancer", "networkloadbalancer"},
			{"sumologic_field_dbidentifier", "dbidentifier"},
			{"sumologic_field_topicname", "topicname"},
			{"sumologic_field_queuename", "queuename"},
			{"sumologic_field_dbclusteridentifier", "dbclusteridentifier"},
			{"sumologic_field_dbinstanceidentifier", "dbinstanceidentifier"},
			{"sumologic_field_apiid", "apiid"},
		}

		passed, failed := 0, 0
		for _, f := range fields {
			id := terraform.Output(t, opts, f.output)
			if id == "" {
				t.Errorf("[field] MISSING: %s (%s)", f.fieldName, f.output)
				failed++
			} else {
				t.Logf("[field] OK: %s id=%s", f.fieldName, id)
				passed++
			}
		}
		t.Logf("[field] %d/%d fields validated", passed, passed+failed)
	})
}

// TestContent_MetricRules deploys and validates NLB, API Gateway, and RDS
// metric rules are created with non-empty trigger names.
func TestContent_MetricRules(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "validate_metric_rules", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)

		rules := []struct {
			output string
			label  string
		}{
			{"sumologic_metric_rule_nlb", "NLB metric rule"},
			{"sumologic_metric_rule_api_gw", "API Gateway metric rule"},
			{"sumologic_metric_rule_rds_cluster", "RDS cluster metric rule"},
			{"sumologic_metric_rule_rds_instance", "RDS instance metric rule"},
		}

		passed, failed := 0, 0
		for _, r := range rules {
			name := terraform.Output(t, opts, r.output)
			if name == "" {
				t.Errorf("[metric-rule] MISSING: %s (%s)", r.label, r.output)
				failed++
			} else {
				t.Logf("[metric-rule] OK: %s name=%s", r.label, name)
				passed++
			}
		}
		t.Logf("[metric-rule] %d/%d metric rules validated", passed, passed+failed)
	})
}
