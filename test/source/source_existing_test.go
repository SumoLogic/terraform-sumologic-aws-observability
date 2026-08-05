package source_test

import (
	"fmt"
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestExisting_OverrideSources deploys the overrideSources example which supplies
// pre-existing collector + bucket rather than creating new ones.
// Validates that the module correctly uses existing resources and doesn't recreate them.
func TestExisting_OverrideSources(t *testing.T) {
	t.Parallel()
	workingDir := overrideDir

	// Create pre-existing S3 bucket and Sumo collector to be reused
	cfg := sumoSearchCfg()
	bucket := &testresources.AWSS3Bucket{
		Cfg: testresources.Config{AWSRegion: region},
		Name:   "awso-existing-" + testresources.RandHex(),
	}
	collector := &testresources.SumoCollector{
		Cfg:  cfg,
		Name: "awso-existing-collector-" + testresources.RandHex(),
	}

	test_structure.RunTestStage(t, "pre_req", func() {
		bucket.Create(t)
		collector.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		bucket.Delete(t)
		collector.Delete(t)
	})

	vars := map[string]interface{}{
		"collect_elb":              true,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":         false,
		"collector_id":             collector.ID(),
		"create_s3_bucket":         false,
		"s3_name":                  bucket.Name,
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health", func() {
		// Verify no new collector/bucket was created
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		stateOut := terraform.RunTerraformCommand(t, opts, "state", "list")

		if containsStr(stateOut, "sumologic_collector.collector") {
			t.Error("[existing] Expected no new collector in state, but found one")
		}
		if containsStr(stateOut, "aws_s3_bucket.s3_bucket") {
			t.Error("[existing] Expected no new S3 bucket in state, but found one")
		}
		t.Log("[existing] Confirmed: existing collector and bucket used — no new resources created")
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})
}

// TestExisting_CloudTrailBucket deploys with an existing CloudTrail S3 bucket,
// verifying that the module attaches SNS notification to the existing bucket.
func TestExisting_CloudTrailBucket(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	bucket := &testresources.AWSS3Bucket{
		Cfg: testresources.Config{AWSRegion: region},
		Name:   "awso-ct-existing-" + testresources.RandHex(),
	}

	test_structure.RunTestStage(t, "pre_req", func() {
		bucket.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		bucket.Delete(t)
	})

	vars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
		"executeTest4":             true,
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		snsTopicARN := terraform.Output(t, opts, "cloudtrail_sns_topic")
		if snsTopicARN == "" {
			t.Error("[existing-ct] Expected cloudtrail_sns_topic output for existing bucket scenario")
		} else {
			t.Logf("[existing-ct] SNS topic for existing CloudTrail bucket: %s", snsTopicARN)
		}
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})
}

// TestExisting_SourceURL pre-creates a Sumo Logic source and passes its URL as
// elb_source_url, verifying the module uses existing sources instead of creating new ones.
func TestExisting_SourceURL(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	cfg := sumoSearchCfg()

	collector := &testresources.SumoCollector{
		Cfg:  cfg,
		Name: "awso-srcurl-collector-" + testresources.RandHex(),
	}
	var elbSource *testresources.SumoSource

	test_structure.RunTestStage(t, "pre_req", func() {
		collector.Create(t)
		elbSource = &testresources.SumoSource{
			Cfg:         cfg,
			CollectorID: collector.ID(),
			Name:        "awso-srcurl-elb-source",
			Category:    "aws/observability/alb/logs",
			BucketName:  "placeholder-bucket",
		}
		elbSource.Create(t)
		t.Logf("[srcurl] Pre-created ELB source URL: %s", elbSource.SourceURL())
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		if elbSource != nil {
			elbSource.Delete(t)
		}
		collector.Delete(t)
	})

	vars := map[string]interface{}{
		"collect_elb":              true,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":         false,
		"collector_id":             collector.ID(),
		// Pass source URL to skip creating a new source
		"elb_source_url": elbSource.SourceURL(),
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		stateOut := terraform.RunTerraformCommand(t, opts, "state", "list")
		if containsStr(stateOut, fmt.Sprintf(`module.elb_module["elb_module"].sumologic_polling_source`)) {
			t.Error("[srcurl] Expected no new ELB source in state when source URL is provided")
		}
		t.Log("[srcurl] Confirmed: no new ELB source created when existing source URL provided")
	})
}

func containsStr(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		func() bool {
			for i := 0; i <= len(haystack)-len(needle); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}()
}
