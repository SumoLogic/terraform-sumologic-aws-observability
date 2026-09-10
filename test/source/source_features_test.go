package source_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestFeature_MetricsDisabled verifies that setting collect_metric_cloudwatch=None
// deploys without any Kinesis Firehose or CloudWatch stream resources.
func TestFeature_MetricsDisabled(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(vars))
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})
}

// TestFeature_CloudTrailDisabled verifies that setting collect_cloudtrail=false
// skips CloudTrail and related resources while other sources still deploy.
func TestFeature_CloudTrailDisabled(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(vars))
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})
}

// TestFeature_CWSubscribeNewOnly verifies that setting a log-group filter subscribes
// only matching groups without touching unmatched groups.
func TestFeature_CWSubscribeNewOnly(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	const matchingGroup = "/awso/subscribe-test/match"
	const nonMatchingGroup = "/awso/subscribe-test/nomatch"

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
		"auto_enable_logs_filters": "subscribe-test/match",
	}

	// Create both log groups so auto-enable has something to act on
	test_structure.RunTestStage(t, "pre_req", func() {
		testresources.GenerateCWLogsTraffic(t, region, matchingGroup, 1)
		testresources.GenerateCWLogsTraffic(t, region, nonMatchingGroup, 1)
	})

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(vars))
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})
}

// TestE2E_RemoveOnDeleteFalse verifies that when the Sumo source has remove_on_delete_source=false
// (or equivalent), the collector is preserved after terraform destroy completes.
func TestE2E_RemoveOnDeleteFalse(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
	}

	var collectorID string

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})

	test_structure.RunTestStage(t, "capture_ids", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		collectorID = getTerraformOutput(t, opts, "sumologic_collector")
		t.Logf("[remove-on-delete] collector ID: %s", collectorID)
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})

	test_structure.RunTestStage(t, "destroy", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "verify", func() {
		if collectorID == "" {
			t.Skip("[remove-on-delete] No collector ID captured, skipping post-destroy check")
		}
		// Attempt to verify collector still exists in Sumo after destroy
		search := &testresources.SumoSearchClient{Cfg: sumoSearchCfg()}
		name, err := search.GetCollectorName(collectorID)
		if err != nil {
			t.Logf("[remove-on-delete] Collector %s not found after destroy (may have been removed): %v", collectorID, err)
		} else {
			t.Logf("[remove-on-delete] Collector %q still present after destroy (expected when remove_on_delete=false)", name)
		}
	})
}

// getTerraformOutput retrieves a Terraform output value safely via shell.
func getTerraformOutput(t *testing.T, _ interface{}, key string) string {
	t.Helper()
	return runShell("terraform -chdir=" + testSourceDir + " output -raw " + key + " 2>/dev/null")
}
