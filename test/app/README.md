# AWSO TF App Module Test Automation

## Pre-requisites

- AWS CLI configured
- Git
- Golang
- Terraform
- Sumo Logic account (preferably dedicated for testing, to avoid conflicts with existing AWSO resources)

## App Module Tests

### Configuration

Set variables in:
- `examples/appmodule/main.auto.tfvars`

Required variables: `sumologic_access_id`, `sumologic_access_key`, `sumologic_organization_id`, `sumologic_environment`, `aws_account_alias`, `sumo_api_endpoint`.

### How to Run

From the repository root:

```bash
# Run all app module tests
go test -v -timeout 120m ./test/app/

# Run a single test
go test -v -timeout 60m ./test/app/ -run ^TestInstall_AllAppsDefault$

# Run all install tests
go test -v -timeout 120m ./test/app/ -run ^TestInstall

# Run all content tests
go test -v -timeout 120m ./test/app/ -run ^TestContent

# Run all lifecycle tests
go test -v -timeout 60m ./test/app/ -run ^TestLifecycle
```

Or use the helper script from this directory:

```bash
./unit_tests.sh                          # run all
./unit_tests.sh TestInstall_Idempotency  # run single
```

Environment overrides:

```bash
SKIP_deploy=true ./unit_tests.sh TestInstall_AllAppsDefault   # skip deploy (reuse saved state)
SKIP_cleanup=true ./unit_tests.sh TestInstall_AllAppsDefault  # skip cleanup (leave infra up)
TEST_TIMEOUT=180m ./unit_tests.sh                             # override timeout
```

### Test Cases

| Test | Description |
|---|---|
| `TestInstall_AllAppsDefault` | Deploys all 15 apps; validates installed_apps output, hierarchy, FERs, fields, metric rules |
| `TestInstall_Idempotency` | Re-applies with no var changes; expects 0 add/destroy |
| `TestInstall_DestroyAndRedeploy` | Full destroy then fresh redeploy from scratch |
| `TestContent_AppsInstalledInCatalog` | Verifies all apps appear in Sumo Logic "Installed Apps" catalog |
| `TestContent_HierarchyStructure` | Verifies AWS Observability hierarchy is created |
| `TestContent_FERsExist` | Validates all 17 Field Extraction Rules |
| `TestContent_FieldsExist` | Validates all 19 Sumo Logic fields |
| `TestContent_MetricRules` | Validates NLB, API GW, and RDS metric rules |
| `TestLifecycle_PlanValidation` | Valid environment passes plan; invalid environment fails with validation error |
| `TestLifecycle_AppSubset` | Deploy with a custom `installation_apps_list` subset; only those apps installed |

### App Installation Location

Apps are installed into the Sumo Logic **"Installed Apps"** catalog via the `sumologic_app` resource — there are no personal or admin folder outputs. Validation uses the `installed_apps` Terraform output which is a map of `app name → {uuid, name, id}`.

### Shared Resources Handling

Org-level Sumo Logic fields may already exist from a previous deployment. The test suite automatically imports them before `apply` (via `importExistingSharedResources`) and removes them from state before `destroy` (via `releaseSharedResources`), so they are never accidentally deleted.

### Failure Recovery

If a test fails partway through, Terraform state is saved under `examples/appmodule/.test-data/`. Re-run with `SKIP_deploy=true` to skip re-deploying and go straight to the failed stage. For full cleanup:

```bash
cd examples/appmodule && terraform destroy
```

### How to Update

- **New app added to `local.tf`**: add its name to `expectedAppNames` in `helpers_test.go`
- **New FER added**: add the new `sumologic_field_extraction_rule_*` output to `TestContent_FERsExist`
- **New field added**: add to `sharedFieldAddrs` in `helpers_test.go` and to `TestContent_FieldsExist`
- **New metric rule**: add to `TestContent_MetricRules`
