package source_test

import (
	"testing"
	"time"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestAutoEnable_Both creates ALBs both BEFORE and AFTER deploy to test:
//   - Pre-existing ALBs: access logs enabled during initial apply
//   - New ALBs (post-deploy): EventBridge rule auto-enables logs when new LB appears
func TestAutoEnable_Both(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              true,
		"collect_classic_lb":       true,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
	}

	// Pre-req: create LBs that exist before deploy
	preALB := &testresources.AWSALB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-pre-alb-" + testresources.RandHex()}
	preCLB := &testresources.AWSClassicLB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-pre-clb-" + testresources.RandHex()}

	test_structure.RunTestStage(t, "pre_req", func() {
		preALB.Create(t)
		preCLB.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		preALB.Delete(t)
		preCLB.Delete(t)
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

	// Post-req: create NEW LBs after deploy to trigger EventBridge auto-enable
	postALB := &testresources.AWSALB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-post-alb-" + testresources.RandHex()}
	postCLB := &testresources.AWSClassicLB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-post-clb-" + testresources.RandHex()}

	test_structure.RunTestStage(t, "post_req", func() {
		postALB.Create(t)
		postCLB.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_postreq", func() {
		postALB.Delete(t)
		postCLB.Delete(t)
	})

	test_structure.RunTestStage(t, "e2e", func() {
		// Wait for EventBridge to fire and auto-enable logs on new LBs
		t.Log("[autoenable] Waiting 2 minutes for EventBridge auto-enable rule to fire...")
		time.Sleep(2 * time.Minute)

		// Validate pre-existing LBs had logs enabled during deploy
		validateALBAccessLogsEnabled(t, preALB.ID())
		validateCLBAccessLogsEnabled(t, preCLB.ID())

		// Validate post-deploy LBs had logs auto-enabled by EventBridge
		validateALBAccessLogsEnabled(t, postALB.ID())
		validateCLBAccessLogsEnabled(t, postCLB.ID())

		// Standard E2E: generate traffic and validate data reaches Sumo
		runStandardE2E(t, vars, workingDir, E2EConfig{
			ALBARN:  postALB.ID(),
			ALBDNS:  postALB.DNS(),
			CLBName: postCLB.ID(),
			CLBDNS:  postCLB.DNS(),
		})
	})
}

// TestAutoEnable_ExistingOnly creates only pre-existing ALB + CLB (no post-deploy LBs) and
// verifies that access logs are enabled on both during the initial apply via the Lambda
// initial-scan. Corresponds to CF's common/alb_auto_enable_existing.yaml and
// common/elb_auto_enable_existing.yaml.
func TestAutoEnable_ExistingOnly(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":               true,
		"collect_classic_lb":        true,
		"collect_cloudtrail":        false,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
	}

	alb := &testresources.AWSALB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-exist-alb-" + testresources.RandHex()}
	clb := &testresources.AWSClassicLB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-exist-clb-" + testresources.RandHex()}

	test_structure.RunTestStage(t, "pre_req", func() {
		alb.Create(t)
		clb.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		alb.Delete(t)
		clb.Delete(t)
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
		validateALBAccessLogsEnabled(t, alb.ID())
		validateCLBAccessLogsEnabled(t, clb.ID())

		runStandardE2E(t, vars, workingDir, E2EConfig{
			ALBARN:  alb.ID(),
			ALBDNS:  alb.DNS(),
			CLBName: clb.ID(),
			CLBDNS:  clb.DNS(),
		})
	})
}

// TestAutoEnable_Mixed deploys with mixed source types (ELB auto-enable + Lambda CW logs)
// and validates that independent source paths work together without conflict.
func TestAutoEnable_Mixed(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              true,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "Lambda Log Forwarder",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
	}

	alb := &testresources.AWSALB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-mix-alb-" + testresources.RandHex()}

	test_structure.RunTestStage(t, "pre_req", func() {
		alb.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		alb.Delete(t)
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

		// Verify Lambda auto-enable subscription filter exists for the log group
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		kfStreamARN := terraform.Output(t, opts, "kf_logs_stream")
		if kfStreamARN != "" {
			validateSubscriptionFilterExists(t, "/awso/e2e/test", kfStreamARN)
		}
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{
			ALBARN: alb.ID(),
			ALBDNS: alb.DNS(),
		})
	})
}
