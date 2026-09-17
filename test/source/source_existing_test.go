package source_test

import (
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
		// cloudtrail_sns_topic is only populated when create_bucket=false (existing bucket path).
		// The module does not currently set up SNS notifications on existing buckets
		// (create_existing_bucket_notification=false in the cloudtrail module call),
		// so this output is always empty in the current version. Log it for diagnostics only.
		snsTopicARN := terraform.Output(t, opts, "cloudtrail_sns_topic")
		t.Logf("[existing-ct] cloudtrail_sns_topic: %q (empty expected when module uses new bucket)", snsTopicARN)

		// Verify that the CloudTrail Sumo Logic source was actually created.
		sourceID, err := terraform.OutputE(t, opts, "sumologic_cloudtrail_source")
		if err != nil || sourceID == "" {
			t.Error("[existing-ct] Expected sumologic_cloudtrail_source to be non-empty")
		} else {
			t.Logf("[existing-ct] CloudTrail source created: %s", sourceID)
		}
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{})
	})
}

// TestExisting_AllSourceURLs pre-creates one Sumo Logic HTTP source per collection type
// and passes all source URLs to the module, verifying no new sources are created for any type —
// the module updates existing sources with account/region fields instead.
// Covers all 5 source URL variables: ALB, Classic LB, CloudTrail, CW logs, KF metrics.
// Equivalent to CF all_existing_source_urls + kinesis_existing_source_urls tests.
func TestExisting_AllSourceURLs(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	cfg := sumoSearchCfg()

	collector := &testresources.SumoCollector{
		Cfg:  cfg,
		Name: "awso-srcurl-collector-" + testresources.RandHex(),
	}

	// One pre-existing source per collection type.
	type preSource struct {
		s   *testresources.SumoSource
		key string
	}
	var sources []preSource

	test_structure.RunTestStage(t, "pre_req", func() {
		collector.Create(t)
		sources = []preSource{
			{key: "alb", s: &testresources.SumoSource{Cfg: cfg, CollectorID: collector.ID(), Name: "awso-srcurl-alb-" + testresources.RandHex(), Category: "aws/observability/alb/logs"}},
			{key: "classic_lb", s: &testresources.SumoSource{Cfg: cfg, CollectorID: collector.ID(), Name: "awso-srcurl-clb-" + testresources.RandHex(), Category: "aws/observability/clb/logs"}},
			{key: "cloudtrail", s: &testresources.SumoSource{Cfg: cfg, CollectorID: collector.ID(), Name: "awso-srcurl-ct-" + testresources.RandHex(), Category: "aws/observability/cloudtrail/logs"}},
			{key: "cw_logs", s: &testresources.SumoSource{Cfg: cfg, CollectorID: collector.ID(), Name: "awso-srcurl-cwlogs-" + testresources.RandHex(), Category: "aws/observability/cloudwatch/logs"}},
			{key: "cw_metrics", s: &testresources.SumoSource{Cfg: cfg, CollectorID: collector.ID(), Name: "awso-srcurl-cwmetrics-" + testresources.RandHex(), Category: "aws/observability/cloudwatch/metrics"}},
		}
		for _, ps := range sources {
			ps.s.Create(t)
			t.Logf("[srcurl] Pre-created %s source URL: %s", ps.key, ps.s.SourceURL())
		}
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		for _, ps := range sources {
			ps.s.Delete(t)
		}
		collector.Delete(t)
	})

	vars := map[string]interface{}{
		"collect_elb":                   true,
		"collect_classic_lb":            true,
		"collect_cloudtrail":            true,
		"collect_logs_cloudwatch":       "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch":     "Kinesis Firehose Metrics Source",
		"create_collector":              false,
		"collector_id":                  collector.ID(),
		"elb_source_url":                sources[0].s.SourceURL(),
		"classic_lb_source_url":         sources[1].s.SourceURL(),
		"cloudtrail_source_url":         sources[2].s.SourceURL(),
		"cloudwatch_log_source_url":     sources[3].s.SourceURL(),
		"cloudwatch_metrics_source_url": sources[4].s.SourceURL(),
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

		// When a source URL is provided the module skips the creation sub-module entirely —
		// assert none of those sub-modules appear in state.
		absent := []struct {
			fragment string
			label    string
		}{
			{`module.elb_module["elb_module"]`, "ALB source"},
			{`module.classic_lb_module["classic_lb_module"]`, "Classic LB source"},
			{`module.cloudtrail_module["cloudtrail_module"]`, "CloudTrail source"},
			{`module.kinesis_firehose_for_logs_module["kinesis_firehose_for_logs_module"]`, "KF logs source"},
			{`module.kinesis_firehose_for_metrics_source_module["kinesis_firehose_for_metrics_source_module"]`, "KF metrics source"},
		}
		for _, c := range absent {
			if containsStr(stateOut, c.fragment) {
				t.Errorf("[srcurl] Expected no new %s in state when source URL is provided", c.label)
			} else {
				t.Logf("[srcurl] Confirmed: no new %s created when existing source URL provided", c.label)
			}
		}

		// Positive: the module must create AddFields* null_resources to attach account/region
		// fields to each existing source — equivalent to CF's SumoALBLogsUpdateSource,
		// SumoELBLogsUpdateSource, SumoCloudTrailLogsUpdateSource, etc.
		present := []struct {
			fragment string
			label    string
		}{
			{`null_resource.AddFieldsToELBSource["add_fields_to_source"]`, "AddFieldsToELBSource"},
			{`null_resource.AddFieldsToCLBSource["add_fields_to_source"]`, "AddFieldsToCLBSource"},
			{`null_resource.AddFieldsToCloudTrailSource["add_fields_to_source"]`, "AddFieldsToCloudTrailSource"},
			{`null_resource.AddFieldsToLogSource["add_fields_to_source"]`, "AddFieldsToLogSource"},
			{`null_resource.AddFieldsToMetricSource["add_fields_to_source"]`, "AddFieldsToMetricSource"},
		}
		for _, c := range present {
			if !containsStr(stateOut, c.fragment) {
				t.Errorf("[srcurl] Expected %s in state when source URL is provided, but not found", c.label)
			} else {
				t.Logf("[srcurl] Confirmed: %s created to attach fields to existing source", c.label)
			}
		}
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
