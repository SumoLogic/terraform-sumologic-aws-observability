package source_test

import (
	"fmt"
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestBucket_SharedPolicyMerge (IT2) deploys CT + ALB + CLB all sharing a single new bucket
// and verifies that the shared bucket policy contains all three service principals — proving
// the module merges permissions correctly rather than one source overwriting another.
func TestBucket_SharedPolicyMerge(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_elb":               true,
		"collect_classic_lb":        true,
		"collect_cloudtrail":        true,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
		"executeTest1":              true, // exposes aws_s3 and aws_iam_role outputs
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
		bucketName := terraform.Output(t, opts, "aws_s3")
		roleName := terraform.Output(t, opts, "aws_iam_role")

		if bucketName == "" {
			t.Fatal("[it2] aws_s3 output is empty — module did not create a shared bucket")
		}
		t.Logf("[it2] shared bucket: %s, IAM role: %s", bucketName, roleName)

		// Shared bucket must carry all three service principals in a single policy
		assertBucketPolicyContains(t, bucketName, "cloudtrail.amazonaws.com")
		assertBucketPolicyContains(t, bucketName, "delivery.logs.amazonaws.com")
		assertBucketPolicyContains(t, bucketName, "logdelivery.elasticloadbalancing.amazonaws.com")

		// S3 → SNS notification must be wired for log delivery
		assertBucketHasSNSNotification(t, bucketName)

		// IAM role must exist for Sumo Logic to read the bucket
		assertIAMRoleExists(t, roleName)
	})
}

// TestBucket_ExistingPolicyAppend (IT5) pre-creates a bucket with a customer-owned policy
// (Sid: CustomerExistingAccess), then deploys CT pointing at that existing bucket. The test
// verifies that the module's Lambda appended the CloudTrail permission rather than replacing
// the customer's original statement.
func TestBucket_ExistingPolicyAppend(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	bucketName := "awso-it5-" + testresources.RandHex()
	bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}

	test_structure.RunTestStage(t, "pre_req", func() {
		bucket.Create(t)
		// Simulate a customer who already has their own bucket policy in place.
		// The module must append to this, not overwrite it.
		customerPolicy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "CustomerExistingAccess",
    "Effect": "Allow",
    "Principal": {"Service": "lambda.amazonaws.com"},
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::%s/*"
  }]
}`, bucketName)
		putBucketPolicy(t, bucketName, customerPolicy)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		bucket.Delete(t)
	})

	vars := map[string]interface{}{
		"collect_elb":               false,
		"collect_classic_lb":        false,
		"collect_cloudtrail":        true,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
		"executeTest4":              true, // exposes cloudtrail_sns_topic output
		"cloudtrail_details": map[string]interface{}{
			"source_name":     "CloudTrail Logs (Region)",
			"source_category": "aws/observability/cloudtrail/logs",
			"description":     "IT5 test: existing bucket append check",
			"bucket_details": map[string]interface{}{
				"create_bucket":        false,
				"create_trail":         false,
				"bucket_name":          bucketName,
				"path_expression":      "AWSLogs/*/CloudTrail/*",
				"force_destroy_bucket": false,
			},
			"fields": map[string]interface{}{},
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
		// Customer's original statement must survive — Lambda must append, not replace
		assertBucketPolicyContains(t, bucketName, "CustomerExistingAccess")
		// CloudTrail permission must have been appended by the module
		assertBucketPolicyContains(t, bucketName, "cloudtrail.amazonaws.com")
		// S3 → SNS notification must be configured on the existing bucket
		assertBucketHasSNSNotification(t, bucketName)
	})
}

// TestBucket_CloudTrailRetentionOnDestroy (IT6) lets the module create a new S3 bucket for
// CloudTrail logs (force_destroy_bucket=false), then runs terraform destroy and confirms the
// bucket AND its objects still exist. A failure means the module deleted a customer's audit data.
//
// Key regression check: verifies that disabled sources' default force_destroy_bucket=true
// does NOT override the active source's force_destroy_bucket=false (the OR-logic bug).
func TestBucket_CloudTrailRetentionOnDestroy(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_cloudtrail":        true,
		"collect_elb":               false,
		"collect_classic_lb":        false,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
		"executeTest1":              true,
		"cloudtrail_details": map[string]interface{}{
			"source_name":     "CloudTrail Logs (Region)",
			"source_category": "aws/observability/cloudtrail/logs",
			"description":     "IT6 test: bucket must survive destroy",
			"bucket_details": map[string]interface{}{
				"create_bucket":        true,
				"create_trail":         true,
				"bucket_name":          "aws-observability-random-id",
				"path_expression":      "AWSLogs/*/CloudTrail/*",
				"force_destroy_bucket": false,
			},
			"fields": map[string]interface{}{},
		},
	}

	const seedKey = "AWSLogs/retention-test/dummy.json"
	var bucketName string

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})

	test_structure.RunTestStage(t, "capture_bucket", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		bucketName = terraform.Output(t, opts, "aws_s3")
		if bucketName == "" {
			t.Fatal("[it6] aws_s3 output is empty — module did not create a bucket")
		}
		t.Logf("[it6] module created bucket: %s", bucketName)
	})

	test_structure.RunTestStage(t, "verify_state", func() {
		assertBucketForceDestroy(t, workingDir, false)
	})

	test_structure.RunTestStage(t, "health", func() {
		assertBucketPolicyContains(t, bucketName, "cloudtrail.amazonaws.com")
		assertBucketHasSNSNotification(t, bucketName)
	})

	test_structure.RunTestStage(t, "seed_bucket", func() {
		if bucketName == "" {
			t.Skip("[it6] bucket name not captured — skipping seed")
			return
		}
		bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		bucket.PutObject(t, seedKey)
	})

	test_structure.RunTestStage(t, "destroy", func() {
		// BucketNotEmpty is the expected outcome — it proves force_destroy=false protects the bucket.
		err := destroyTerraformAllowingErrors(t, workingDir)
		if err != nil {
			t.Logf("[it6] terraform destroy returned error (expected — bucket is protected): %v", err)
		}
	})

	test_structure.RunTestStage(t, "verify_retention", func() {
		if bucketName == "" {
			t.Skip("[it6] bucket name not captured — skipping retention check")
			return
		}
		bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		bucket.AssertExists(t)
		bucket.AssertObjectExists(t, seedKey)
		t.Log("[it6] bucket and objects survived terraform destroy — customer audit logs are safe")
	})

	defer test_structure.RunTestStage(t, "cleanup_bucket", func() {
		if bucketName == "" {
			return
		}
		b := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		b.ForceDelete(t)
	})
}

// TestBucket_CloudTrailForceDestroyCleanup validates the opposite path: when a user sets
// force_destroy_bucket=true, terraform destroy should fully remove the bucket — even if it
// contains objects. This confirms the cleanup path still works after the force_destroy fix.
func TestBucket_CloudTrailForceDestroyCleanup(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_cloudtrail":        true,
		"collect_elb":               false,
		"collect_classic_lb":        false,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
		"executeTest1":              true,
		"cloudtrail_details": map[string]interface{}{
			"source_name":     "CloudTrail Logs (Region)",
			"source_category": "aws/observability/cloudtrail/logs",
			"description":     "Force-destroy cleanup test: bucket must be deleted",
			"bucket_details": map[string]interface{}{
				"create_bucket":        true,
				"create_trail":         true,
				"bucket_name":          "aws-observability-random-id",
				"path_expression":      "AWSLogs/*/CloudTrail/*",
				"force_destroy_bucket": true,
			},
			"fields": map[string]interface{}{},
		},
	}

	var bucketName string

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})

	test_structure.RunTestStage(t, "capture_bucket", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		bucketName = terraform.Output(t, opts, "aws_s3")
		if bucketName == "" {
			t.Fatal("[force-destroy] aws_s3 output is empty — module did not create a bucket")
		}
		t.Logf("[force-destroy] module created bucket: %s", bucketName)
	})

	test_structure.RunTestStage(t, "verify_state", func() {
		assertBucketForceDestroy(t, workingDir, true)
	})

	test_structure.RunTestStage(t, "seed_bucket", func() {
		if bucketName == "" {
			t.Skip("[force-destroy] bucket name not captured — skipping seed")
			return
		}
		bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		bucket.PutObject(t, "AWSLogs/cleanup-test/dummy.json")
	})

	test_structure.RunTestStage(t, "destroy", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "verify_deleted", func() {
		if bucketName == "" {
			t.Skip("[force-destroy] bucket name not captured — skipping deletion check")
			return
		}
		bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		bucket.AssertNotExists(t)
		t.Log("[force-destroy] bucket correctly deleted on destroy — cleanup path works")
	})
}

// TestBucket_SharedMixedForceDestroy deploys CT + ALB sharing one bucket where CloudTrail
// sets force_destroy=false and ALB sets force_destroy=true. The bucket must be protected
// because ANY source saying "false" should win — one source protecting the bucket is enough.
func TestBucket_SharedMixedForceDestroy(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	vars := map[string]interface{}{
		"collect_cloudtrail":        true,
		"collect_elb":               true,
		"collect_classic_lb":        false,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
		"executeTest1":              true,
		"cloudtrail_details": map[string]interface{}{
			"source_name":     "CloudTrail Logs (Region)",
			"source_category": "aws/observability/cloudtrail/logs",
			"description":     "Mixed force_destroy test: CT says false",
			"bucket_details": map[string]interface{}{
				"create_bucket":        true,
				"create_trail":         true,
				"bucket_name":          "aws-observability-random-id",
				"path_expression":      "AWSLogs/*/CloudTrail/*",
				"force_destroy_bucket": false,
			},
			"fields": map[string]interface{}{},
		},
		"elb_details": map[string]interface{}{
			"source_name":     "Elb Logs (Region)",
			"source_category": "aws/observability/alb/logs",
			"description":     "Mixed force_destroy test: ALB says true",
			"bucket_details": map[string]interface{}{
				"create_bucket":        true,
				"bucket_name":          "aws-observability-random-id",
				"path_expression":      "*elasticloadbalancing/AWSLogs/*",
				"force_destroy_bucket": true,
			},
			"fields": map[string]interface{}{},
		},
	}

	const seedKey = "AWSLogs/mixed-test/dummy.json"
	var bucketName string

	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployTerraform(t, workingDir, vars, "")
		testresources.AssertResourceCounts(t, counts)
	})

	test_structure.RunTestStage(t, "capture_bucket", func() {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		bucketName = terraform.Output(t, opts, "aws_s3")
		if bucketName == "" {
			t.Fatal("[mixed] aws_s3 output is empty — module did not create a shared bucket")
		}
		t.Logf("[mixed] shared bucket: %s", bucketName)
	})

	test_structure.RunTestStage(t, "verify_state", func() {
		// CT says false → bucket must be protected regardless of ALB saying true
		assertBucketForceDestroy(t, workingDir, false)
	})

	test_structure.RunTestStage(t, "seed_bucket", func() {
		if bucketName == "" {
			t.Skip("[mixed] bucket name not captured — skipping seed")
			return
		}
		bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		bucket.PutObject(t, seedKey)
	})

	test_structure.RunTestStage(t, "destroy", func() {
		destroyTerraform(t, workingDir)
	})

	test_structure.RunTestStage(t, "verify_retention", func() {
		if bucketName == "" {
			t.Skip("[mixed] bucket name not captured — skipping retention check")
			return
		}
		bucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		bucket.AssertExists(t)
		bucket.AssertObjectExists(t, seedKey)
		t.Log("[mixed] bucket survived destroy — one source saying false protects the shared bucket")
	})

	defer test_structure.RunTestStage(t, "cleanup_bucket", func() {
		if bucketName == "" {
			return
		}
		b := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: bucketName}
		b.ForceDelete(t)
	})
}

// TestBucket_AllExisting (IT7) pre-creates three separate buckets — one per source — and
// deploys with all three sources pointing to their respective pre-existing buckets. Every
// bucket policy update goes through the Lambda path. The test checks each bucket independently
// received the correct service principal and SNS notification.
func TestBucket_AllExisting(t *testing.T) {
	t.Parallel()
	workingDir := testSourceDir

	ctBucketName  := "awso-it7-ct-"  + testresources.RandHex()
	albBucketName := "awso-it7-alb-" + testresources.RandHex()
	clbBucketName := "awso-it7-clb-" + testresources.RandHex()

	ctBucket  := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: ctBucketName}
	albBucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: albBucketName}
	clbBucket := &testresources.AWSS3Bucket{Cfg: awsCfg(), Name: clbBucketName}

	test_structure.RunTestStage(t, "pre_req", func() {
		ctBucket.Create(t)
		albBucket.Create(t)
		clbBucket.Create(t)
	})
	defer test_structure.RunTestStage(t, "cleanup_prereq", func() {
		ctBucket.Delete(t)
		albBucket.Delete(t)
		clbBucket.Delete(t)
	})

	vars := map[string]interface{}{
		"collect_elb":               true,
		"collect_classic_lb":        true,
		"collect_cloudtrail":        true,
		"collect_logs_cloudwatch":   "None",
		"collect_metric_cloudwatch": "None",
		"create_collector":          true,
		"executeTest4":              true, // exposes per-source SNS topic outputs
		"cloudtrail_details": map[string]interface{}{
			"source_name":     "CloudTrail Logs (Region)",
			"source_category": "aws/observability/cloudtrail/logs",
			"description":     "IT7 test: existing CT bucket",
			"bucket_details": map[string]interface{}{
				"create_bucket":        false,
				"create_trail":         false,
				"bucket_name":          ctBucketName,
				"path_expression":      "AWSLogs/*/CloudTrail/*",
				"force_destroy_bucket": false,
			},
			"fields": map[string]interface{}{},
		},
		"elb_details": map[string]interface{}{
			"source_name":     "Elb Logs (Region)",
			"source_category": "aws/observability/alb/logs",
			"description":     "IT7 test: existing ALB bucket",
			"bucket_details": map[string]interface{}{
				"create_bucket":        false,
				"bucket_name":          albBucketName,
				"path_expression":      "*elasticloadbalancing/AWSLogs/*",
				"force_destroy_bucket": false,
			},
			"fields": map[string]interface{}{},
		},
		"classic_lb_details": map[string]interface{}{
			"source_name":     "Classic lb Logs (Region)",
			"source_category": "aws/observability/clb/logs",
			"description":     "IT7 test: existing CLB bucket",
			"bucket_details": map[string]interface{}{
				"create_bucket":        false,
				"bucket_name":          clbBucketName,
				"path_expression":      "*classicloadbalancing/AWSLogs/*",
				"force_destroy_bucket": false,
			},
			"fields": map[string]interface{}{},
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
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		roleName := terraform.Output(t, opts, "aws_iam_role")

		// Each bucket must carry only its own service principal (Lambda ran independently)
		assertBucketPolicyContains(t, ctBucketName, "cloudtrail.amazonaws.com")
		assertBucketPolicyContains(t, albBucketName, "delivery.logs.amazonaws.com")
		assertBucketPolicyContains(t, clbBucketName, "logdelivery.elasticloadbalancing.amazonaws.com")

		// All three buckets must have S3 → SNS notification wired
		assertBucketHasSNSNotification(t, ctBucketName)
		assertBucketHasSNSNotification(t, albBucketName)
		assertBucketHasSNSNotification(t, clbBucketName)

		// Shared IAM role for Sumo Logic must exist
		assertIAMRoleExists(t, roleName)
	})
}
