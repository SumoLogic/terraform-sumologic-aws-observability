package app_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestInstall_AllAppsDefault deploys the app module with all defaults and validates
// that all apps are installed into the Sumo Logic "Installed Apps" catalog,
// along with hierarchy, FERs, and fields.
func TestInstall_AllAppsDefault(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "health", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)

		// Validate all apps appear in the installed_apps output
		validateInstalledApps(t, workingDir, expectedAppNames)

		// Hierarchy
		validateHierarchyExists(t, opts)

		// FERs
		for _, out := range []string{
			"sumologic_field_extraction_rule_apigateway",
			"sumologic_field_extraction_rule_apigateway_access_logs",
			"sumologic_field_extraction_rule_alb",
			"sumologic_field_extraction_rule_alb_cloudtrail",
			"sumologic_field_extraction_rule_elb",
			"sumologic_field_extraction_rule_clb_cloudtrail",
			"sumologic_field_extraction_rule_nlb_cloudtrail",
			"sumologic_field_extraction_rule_dynamodb",
			"sumologic_field_extraction_rule_elasticache",
			"sumologic_field_extraction_rule_ecs",
			"sumologic_field_extraction_rule_ec2metrics",
			"sumologic_field_extraction_rule_lambda",
			"sumologic_field_extraction_rule_lambda_cw",
			"sumologic_field_extraction_rule_rds",
			"sumologic_field_extraction_rule_cw",
			"sumologic_field_extraction_rule_sns",
			"sumologic_field_extraction_rule_sqs",
		} {
			validateFERExists(t, opts, out)
		}

		// Fields
		for _, out := range []string{
			"sumologic_field_account", "sumologic_field_region",
			"sumologic_field_accountid", "sumologic_field_namespace",
			"sumologic_field_loadbalancer", "sumologic_field_loadbalancername",
			"sumologic_field_apiname", "sumologic_field_tablename",
			"sumologic_field_instanceid", "sumologic_field_clustername",
			"sumologic_field_cacheclusterid", "sumologic_field_functionname",
			"sumologic_field_networkloadbalancer", "sumologic_field_dbidentifier",
			"sumologic_field_topicname", "sumologic_field_queuename",
			"sumologic_field_dbclusteridentifier", "sumologic_field_dbinstanceidentifier",
			"sumologic_field_apiid",
		} {
			validateFieldExists(t, opts, out)
		}

		// Metric rules
		for _, out := range []string{
			"sumologic_metric_rule_nlb", "sumologic_metric_rule_api_gw",
			"sumologic_metric_rule_rds_cluster", "sumologic_metric_rule_rds_instance",
		} {
			val := terraform.Output(t, opts, out)
			if val == "" {
				t.Errorf("[metric-rule] %s: empty", out)
			} else {
				t.Logf("[metric-rule] %s: %s (OK)", out, val)
			}
		}
	})
}

// TestInstall_Idempotency deploys, then re-applies with no var changes and asserts
// the second apply produces 0 adds and 0 destroys.
func TestInstall_Idempotency(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "health", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		validateInstalledApps(t, workingDir, expectedAppNames)
		validateHierarchyExists(t, opts)
	})

	test_structure.RunTestStage(t, "idempotency", func() {
		counts := redeployApp(t, map[string]interface{}{})
		if counts.Add != 0 {
			t.Errorf("[idempotency] Expected 0 additions on re-apply, got %d", counts.Add)
		}
		if counts.Destroy != 0 {
			t.Errorf("[idempotency] Expected 0 destroys on re-apply, got %d", counts.Destroy)
		}
		t.Logf("[idempotency] Re-apply: %d added, %d changed, %d destroyed",
			counts.Add, counts.Change, counts.Destroy)
	})
}

// TestInstall_DestroyAndRedeploy does a full deploy → destroy → redeploy cycle
// to verify fresh reinstall works cleanly without stale state conflicts.
func TestInstall_DestroyAndRedeploy(t *testing.T) {
	t.Parallel()

	test_structure.RunTestStage(t, "first_deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})

	test_structure.RunTestStage(t, "first_destroy", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "second_deploy", func() {
		counts := deployApp(t, map[string]interface{}{})
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})

	test_structure.RunTestStage(t, "health_after_redeploy", func() {
		opts := test_structure.LoadTerraformOptions(t, appModuleDir)
		validateInstalledApps(t, appModuleDir, expectedAppNames)
		validateHierarchyExists(t, opts)
	})
}
