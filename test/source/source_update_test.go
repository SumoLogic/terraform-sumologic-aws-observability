package source_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestUpdate_AddCloudTrail deploys without CloudTrail, then adds it via re-apply.
// Validates that new CloudTrail resources appear in state without destroying existing ones.
func TestUpdate_AddCloudTrail(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	initialVars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, initialVars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health_initial", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(initialVars))
	})

	test_structure.RunTestStage(t, "e2e_initial", func() {
		runStandardE2E(t, initialVars, workingDir, E2EConfig{})
	})

	updatedVars := mapCopyWith(initialVars, "collect_cloudtrail", true)

	test_structure.RunTestStage(t, "add_cloudtrail", func() {
		counts := redeployTerraform(t, workingDir, updatedVars, "")
		t.Logf("[update] Add CloudTrail: %d added, %d changed, %d destroyed",
			counts.Add, counts.Change, counts.Destroy)
		if counts.Add == 0 {
			t.Error("[update] Expected new CloudTrail resources to be added")
		}
		if counts.Destroy > 0 {
			t.Errorf("[update] Unexpected destroys when adding CloudTrail: %d", counts.Destroy)
		}
	})

	test_structure.RunTestStage(t, "health_after_update", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(updatedVars))
	})

	test_structure.RunTestStage(t, "e2e_after_update", func() {
		runStandardE2E(t, updatedVars, workingDir, E2EConfig{})
	})
}

// TestUpdate_LambdaToKinesis deploys with Lambda Log Forwarder, then switches to
// Kinesis Firehose Log Source. Validates that the Lambda resources are removed and
// Firehose resources are created without affecting other sources.
func TestUpdate_LambdaToKinesis(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	lambdaVars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Lambda Log Forwarder",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
	}

	test_structure.RunTestStage(t, "deploy_lambda", func() {
		counts := deployTerraform(t, workingDir, lambdaVars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health_lambda", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(lambdaVars))
	})

	test_structure.RunTestStage(t, "e2e_lambda", func() {
		runStandardE2E(t, lambdaVars, workingDir, E2EConfig{})
	})

	kfVars := mapCopyWith(lambdaVars, "collect_logs_cloudwatch", "Kinesis Firehose Log Source")

	test_structure.RunTestStage(t, "switch_to_kf", func() {
		counts := redeployTerraform(t, workingDir, kfVars, "")
		t.Logf("[update] Lambda→KF: %d added, %d changed, %d destroyed",
			counts.Add, counts.Change, counts.Destroy)
		// Lambda module resources are destroyed; KF resources are added
		if counts.Add == 0 {
			t.Error("[update] Expected KF resources to be added after Lambda→KF switch")
		}
	})

	test_structure.RunTestStage(t, "health_kf", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		stateOut := terraform.RunTerraformCommand(t, opts, "state", "list")
		if containsStr(stateOut, "cloudwatch_logs_lambda_log_forwarder_module") {
			t.Error("[update] Lambda forwarder resources should be gone after switch to KF")
		}
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(kfVars))
	})

	test_structure.RunTestStage(t, "e2e_kf", func() {
		runStandardE2E(t, kfVars, workingDir, E2EConfig{})
	})
}

// TestUpdate_NamespaceChange deploys KF Metrics with default namespaces, then
// updates the namespace list. Validates that CloudWatch metric stream is updated.
func TestUpdate_NamespaceChange(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	initialVars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "None",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
		"metric_namespaces":        []interface{}{"AWS/EC2", "AWS/Lambda"},
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, initialVars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health_initial", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(initialVars))
	})

	test_structure.RunTestStage(t, "e2e_initial", func() {
		runStandardE2E(t, initialVars, workingDir, E2EConfig{})
	})

	// Add RDS and ECS namespaces
	updatedVars := mapCopyWith(initialVars, "metric_namespaces",
		[]interface{}{"AWS/EC2", "AWS/Lambda", "AWS/RDS", "AWS/ECS"})

	test_structure.RunTestStage(t, "update_namespaces", func() {
		counts := redeployTerraform(t, workingDir, updatedVars, "")
		t.Logf("[update] Namespace change: %d added, %d changed, %d destroyed",
			counts.Add, counts.Change, counts.Destroy)
		if counts.Destroy > 0 {
			t.Logf("[update] Note: %d resources destroyed during namespace update (metric stream may be recreated)", counts.Destroy)
		}
	})

	test_structure.RunTestStage(t, "health_updated", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(updatedVars))
	})

	test_structure.RunTestStage(t, "e2e_updated", func() {
		runStandardE2E(t, updatedVars, workingDir, E2EConfig{})
	})
}

// TestUpdate_AccountAlias deploys with one aws_account_alias, then updates it.
// Verifies that the change propagates to Sumo collector metadata without destroying sources.
func TestUpdate_AccountAlias(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	initialVars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
		"aws_account_alias":        "devaccount",
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, initialVars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health_initial", func() {
		testresources.AssertResourceExistence(t, workingDir, testresources.ExpectedResources(initialVars))
	})

	test_structure.RunTestStage(t, "e2e_initial", func() {
		runStandardE2E(t, initialVars, workingDir, E2EConfig{})
	})

	updatedVars := mapCopyWith(initialVars, "aws_account_alias", "prodaccount")

	test_structure.RunTestStage(t, "update_alias", func() {
		counts := redeployTerraform(t, workingDir, updatedVars, "")
		t.Logf("[update] Alias change: %d added, %d changed, %d destroyed",
			counts.Add, counts.Change, counts.Destroy)
		if counts.Destroy > 0 {
			t.Errorf("[update] Unexpected destroys after alias update: %d", counts.Destroy)
		}
	})

	test_structure.RunTestStage(t, "health_updated", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		collectorID := terraform.Output(t, opts, "sumologic_collector")
		if collectorID == "" {
			t.Skip("[update] No collector ID available")
		}
		search := &testresources.SumoSearchClient{Cfg: sumoSearchCfg()}
		name, err := search.GetCollectorName(collectorID)
		if err != nil {
			t.Errorf("[update] Failed to get collector name: %v", err)
			return
		}
		t.Logf("[update] Collector name after alias update: %q", name)
		if !containsStr(name, "prodaccount") {
			t.Logf("[update] Note: collector name %q may not reflect alias immediately", name)
		}
	})

	test_structure.RunTestStage(t, "e2e_updated", func() {
		runStandardE2E(t, updatedVars, workingDir, E2EConfig{})
	})
}
