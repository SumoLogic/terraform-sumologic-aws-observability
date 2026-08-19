# AWSO TF Source Module Test Automation

## Pre-requisites

- AWS CLI configured
- Git
- Golang
- Terraform
- Sumo Logic account (preferably dedicated for testing, to avoid conflicts with existing AWSO resources)

## Source Module Tests

### Configuration

Set variables in:
- `examples/sourcemodule/testSource/main.auto.tfvars` — used by most tests
- `examples/sourcemodule/overrideSources/main.auto.tfvars` — used by `TestExisting_OverrideSources`

Required variables: `sumologic_access_id`, `sumologic_access_key`, `sumologic_organization_id`, `sumologic_environment`, `aws_account_alias`, `sumo_api_endpoint`.

### How to Run

From the repository root:

```bash
# Run all source module tests
go test -v -timeout 180m ./test/source/

# Run a single test
go test -v -timeout 60m ./test/source/ -run ^TestBasic_AllDefaults$

# Run all basic tests
go test -v -timeout 120m ./test/source/ -run ^TestBasic

# Run all autoenable tests
go test -v -timeout 120m ./test/source/ -run ^TestAutoEnable

# Run all update tests
go test -v -timeout 120m ./test/source/ -run ^TestUpdate
```

Or use the helper script from this directory:

```bash
./unit_tests.sh                         # run all
./unit_tests.sh TestBasic_AllDefaults   # run single
```

Environment overrides:

```bash
SKIP_deploy=true ./unit_tests.sh TestBasic_AllDefaults   # skip deploy (reuse saved state)
SKIP_cleanup=true ./unit_tests.sh TestBasic_AllDefaults  # skip cleanup (leave infra up)
TEST_TIMEOUT=240m ./unit_tests.sh                        # override timeout
```

### Test Cases

| File | Test | Description |
|---|---|---|
| `source_basic_test.go` | `TestBasic_AllDefaults` | All sources enabled with defaults; full E2E data flow validation |
| | `TestBasic_NothingInstalled` | All sources disabled; collector + IAM + fields only |
| | `TestBasic_LambdaForwarder` | Lambda Log Forwarder instead of KF Logs; E2E via Lambda path |
| | `TestBasic_LambdaForwarderTagFilter` | Lambda Forwarder with tag-based auto-enable filter |
| | `TestBasic_KinesisWithTagFilter` | KF Logs + KF Metrics with metrics tag filter |
| `source_autoenable_test.go` | `TestAutoEnable_Both` | Pre-existing + post-deploy LBs; validates EventBridge auto-enable |
| | `TestAutoEnable_ExistingOnly` | Only pre-existing LBs; validates access logs enabled at deploy |
| | `TestAutoEnable_NewOnly` | Only post-deploy LBs; validates EventBridge fires for new LBs |
| | `TestAutoEnable_Mixed` | ELB auto-enable + Lambda CW logs together |
| `source_existing_test.go` | `TestExisting_OverrideSources` | Pre-existing collector + bucket; no new resources created |
| | `TestExisting_CloudTrailBucket` | Existing CloudTrail bucket; SNS attached without creating new bucket |
| | `TestExisting_SourceURL` | Pre-created Sumo source URL passed as variable |
| `source_features_test.go` | `TestFeature_BucketRetention` | `force_destroy=false`; bucket survives `terraform destroy` |
| | `TestFeature_MetricsDisabled` | `collect_metric_cloudwatch=None`; no KF/CW stream resources |
| | `TestFeature_CloudTrailDisabled` | `collect_cloudtrail=false`; other sources unaffected |
| | `TestFeature_CWSubscribeNewOnly` | Log group filter subscribes only matching groups |
| | `TestE2E_RemoveOnDeleteFalse` | Collector persists after destroy when `remove_on_delete=false` |
| `source_tags_test.go` | `TestTags_Propagation` | `aws_resource_tags` propagated to S3, IAM, KF, CloudTrail |
| | `TestTags_UpdateExisting` | Tag update re-applied without destroying resources |
| | `TestTags_RemoveTags` | Removed tag keys absent after re-apply |
| `source_update_test.go` | `TestUpdate_AddCloudTrail` | Add CloudTrail to existing deployment; no destroys |
| | `TestUpdate_LambdaToKinesis` | Switch Lambda→KF Logs; Lambda removed, KF added |
| | `TestUpdate_NamespaceChange` | Add/remove CloudWatch metric namespaces |
| | `TestUpdate_AccountAlias` | Update `aws_account_alias`; no unexpected destroys |

### E2E Validation

Every test that deploys infrastructure includes an `e2e` stage that:
1. Validates ALB/CLB access logging is enabled (polls 30s intervals, 120s timeout)
2. Generates traffic (HTTP requests to LBs, CloudTrail API calls, CW log events)
3. Waits for S3-based delivery pipelines (6 minutes)
4. Queries Sumo Logic for data using the Search Job API / Metrics API with retry schedule: **30s → 60s → 2m → 2m → 2m**

### Failure Recovery

If a test fails partway through, Terraform state is saved under the example directory's `.test-data/`. Re-run with `SKIP_deploy=true` to resume from after the deploy stage. For full cleanup:

```bash
cd examples/sourcemodule/testSource && terraform destroy
cd examples/sourcemodule/overrideSources && terraform destroy
```

### How to Update

- **New source type added**: add its output key to `sourceOutputKeys` in `helpers_test.go` and add expected state addresses to `expected_resources.go`
- **New AWS resource tagged**: add tag validation call in `source_tags_test.go`
- **New variable combination**: add a test function in the appropriate `source_*_test.go` file following the `deploy → health → e2e → cleanup` stage pattern
