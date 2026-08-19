package source_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestBasic_AllDefaults deploys the source module with all defaults enabled:
// ALB + CLB + CloudTrail + KF Logs + KF Metrics, then validates E2E data flow.
func TestBasic_AllDefaults(t *testing.T) {
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

	// Pre-requisites: create LBs before deploy so they exist when Terraform enables access logs
	alb := &testresources.AWSALB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-e2e-alb-" + testresources.RandHex()}
	clb := &testresources.AWSClassicLB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-e2e-clb-" + testresources.RandHex()}

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
		runStandardE2E(t, vars, workingDir, E2EConfig{
			ALBARN:  alb.ID(),
			ALBDNS:  alb.DNS(),
			CLBName: clb.ID(),
			CLBDNS:  clb.DNS(),
		})
	})
}

// TestBasic_NothingInstalled deploys with all sources disabled to verify a minimal
// (collector + IAM + fields only) deployment succeeds without errors.
func TestBasic_NothingInstalled(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "None",
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
}

// TestBasic_LambdaForwarder deploys with Lambda Log Forwarder instead of KF Logs,
// then validates CW logs reach Sumo through the Lambda path.
func TestBasic_LambdaForwarder(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Lambda Log Forwarder",
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

// TestBasic_LambdaForwarderTagFilter deploys Lambda Forwarder with a tag-based
// auto-enable filter and verifies only matching log groups get subscription filters.
func TestBasic_LambdaForwarderTagFilter(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":                   false,
		"collect_classic_lb":            false,
		"collect_cloudtrail":            false,
		"collect_logs_cloudwatch":       "Lambda Log Forwarder",
		"collect_metric_cloudwatch":     "None",
		"create_collector":              true,
		"auto_enable_logs_filters":      "awso-tag-test",
		"auto_enable_logs_tags_filters": "environment=test",
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

// TestBasic_KinesisWithTagFilter deploys KF Logs with a metrics tag filter,
// validating that the tag filter config doesn't break the deployment.
func TestBasic_KinesisWithTagFilter(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
		"metrics_tag_filters": []interface{}{
			map[string]interface{}{
				"type":      "TagFilters",
				"namespace": "AWS/EC2",
				"tags":      []interface{}{"environment=test"},
			},
		},
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
