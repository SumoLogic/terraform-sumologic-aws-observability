package source_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestTags_Propagation deploys with aws_resource_tags and verifies that all
// AWS resources (S3, IAM role, Firehose streams, CloudTrail) carry the expected tags.
func TestTags_Propagation(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	expectedTags := map[string]string{
		"environment": "test",
		"team":        "observability",
		"project":     "awso",
	}

	vars := map[string]interface{}{
		"collect_elb":              true,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
		"aws_resource_tags":        expectedTags,
	}

	alb := &testresources.AWSALB{Cfg: testresources.Config{AWSRegion: region}, Name: "awso-tags-alb-" + testresources.RandHex()}

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
	})

	test_structure.RunTestStage(t, "validate_tags", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)

		// Validate S3 bucket tags
		bucketName := terraform.Output(t, opts, "aws_s3")
		if bucketName != "" {
			actualTags := validateS3BucketTags(t, bucketName, expectedTags)
			t.Logf("[tags] S3 bucket %s tags: %v", bucketName, actualTags)
		}

		// Validate IAM role tags
		roleName := terraform.Output(t, opts, "aws_iam_role")
		if roleName != "" {
			actualTags := validateIAMRoleTags(t, roleName, expectedTags)
			t.Logf("[tags] IAM role %s tags: %v", roleName, actualTags)
		}

		// Validate KF logs stream tags
		kfLogsStream := terraform.Output(t, opts, "kf_logs_stream")
		if kfLogsStream != "" {
			validateKinesisFirehoseTags(t, kfLogsStream, expectedTags)
			t.Logf("[tags] KF logs stream %s tags validated", kfLogsStream)
		}

		// Validate KF metrics stream tags
		kfMetricsStream := terraform.Output(t, opts, "kf_metrics_stream")
		if kfMetricsStream != "" {
			validateKinesisFirehoseTags(t, kfMetricsStream, expectedTags)
			t.Logf("[tags] KF metrics stream %s tags validated", kfMetricsStream)
		}

		// Validate CloudTrail tags
		trailName := terraform.Output(t, opts, "aws_cloudtrail_name")
		if trailName != "" {
			validateCloudTrailTags(t, trailName, expectedTags)
			t.Logf("[tags] CloudTrail %s tags validated", trailName)
		}
	})

	test_structure.RunTestStage(t, "e2e", func() {
		runStandardE2E(t, vars, workingDir, E2EConfig{
			ALBARN: alb.ID(),
			ALBDNS: alb.DNS(),
		})
	})
}

// TestTags_UpdateExisting verifies that updating aws_resource_tags on an
// existing deployment propagates tag changes to all AWS resources via re-apply.
func TestTags_UpdateExisting(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	initialTags := map[string]string{
		"env":   "staging",
		"owner": "team-a",
	}
	updatedTags := map[string]string{
		"env":   "production",
		"owner": "team-b",
		"cost":  "shared",
	}

	initialVars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       true,
		"collect_logs_cloudwatch":  "None",
		"collect_metric_cloudwatch": "Kinesis Firehose Metrics Source",
		"create_collector":         true,
		"aws_resource_tags":        initialTags,
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

	updatedVars := mapCopyWith(initialVars, "aws_resource_tags", updatedTags)

	test_structure.RunTestStage(t, "update", func() {
		counts := redeployTerraform(t, workingDir, updatedVars, "")
		t.Logf("[tags-update] Re-apply: %d added, %d changed, %d destroyed",
			counts.Add, counts.Change, counts.Destroy)
		if counts.Destroy > 0 {
			t.Errorf("[tags-update] Unexpected destroys during tag update: %d", counts.Destroy)
		}
	})

	test_structure.RunTestStage(t, "validate_updated_tags", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)

		bucketName := terraform.Output(t, opts, "aws_s3")
		if bucketName != "" {
			validateS3BucketTags(t, bucketName, updatedTags)
		}

		kfMetricsStream := terraform.Output(t, opts, "kf_metrics_stream")
		if kfMetricsStream != "" {
			validateKinesisFirehoseTags(t, kfMetricsStream, updatedTags)
		}

		trailName := terraform.Output(t, opts, "aws_cloudtrail_name")
		if trailName != "" {
			validateCloudTrailTags(t, trailName, updatedTags)
		}
	})
}

// TestTags_RemoveTags verifies that removing keys from aws_resource_tags and re-applying
// causes those tag keys to disappear from all AWS resources.
func TestTags_RemoveTags(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	initialTags := map[string]string{
		"keep-me":   "yes",
		"remove-me": "no",
	}

	initialVars := map[string]interface{}{
		"collect_elb":              false,
		"collect_classic_lb":       false,
		"collect_cloudtrail":       false,
		"collect_logs_cloudwatch":  "Kinesis Firehose Log Source",
		"collect_metric_cloudwatch": "None",
		"create_collector":         true,
		"aws_resource_tags":        initialTags,
	}

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, initialVars, "")
		testresources.AssertResourceCounts(t, counts)
	})
	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "health_initial", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		kfLogsStream := terraform.Output(t, opts, "kf_logs_stream")
		if kfLogsStream != "" {
			validateKinesisFirehoseTags(t, kfLogsStream, initialTags)
		}
	})

	// Remove "remove-me" tag
	prunedTags := map[string]string{"keep-me": "yes"}
	updatedVars := mapCopyWith(initialVars, "aws_resource_tags", prunedTags)

	test_structure.RunTestStage(t, "remove_tags", func() {
		counts := redeployTerraform(t, workingDir, updatedVars, "")
		if counts.Destroy > 0 {
			t.Errorf("[tags-remove] Unexpected destroys: %d", counts.Destroy)
		}
	})

	test_structure.RunTestStage(t, "validate_removed_tags", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		kfLogsStream := terraform.Output(t, opts, "kf_logs_stream")
		if kfLogsStream != "" {
			actual := validateS3BucketTags(t, kfLogsStream, prunedTags)
			testresources.AssertTagsAbsent(t, "kf_logs_stream:"+kfLogsStream, actual, []string{"remove-me"})
		}
	})
}

// mapCopyWith returns a shallow copy of m with key set to val.
func mapCopyWith(m map[string]interface{}, key string, val interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out[key] = val
	return out
}
