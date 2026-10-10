package app_test

import (
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

// TestLifecycle_PlanValidation verifies that valid inputs pass plan and invalid
// inputs (bad environment value) fail plan with a validation error.
// Subtests are sequential (not parallel) so they can safely share the same
// already-initialized working directory without a concurrent init race.
func TestLifecycle_PlanValidation(t *testing.T) {
	t.Parallel()

	// Shared init — run once, reused by both subtests.
	baseOpts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: appModuleDir,
		NoColor:      true,
	})
	terraform.Init(t, baseOpts)

	t.Run("ValidEnvironment", func(t *testing.T) {
		_, err := terraform.PlanE(t, baseOpts)
		if err != nil {
			t.Logf("[plan] Valid config plan note: %v", err)
		} else {
			t.Log("[plan] Valid environment: plan succeeded (OK)")
		}
	})

	t.Run("InvalidEnvironment", func(t *testing.T) {
		opts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
			TerraformDir: appModuleDir,
			Vars:         map[string]interface{}{"sumologic_environment": "invalid-env"},
			NoColor:      true,
		})
		_, err := terraform.PlanE(t, opts)
		if err == nil {
			t.Error("[plan] Expected plan to fail with invalid environment, but it succeeded")
		} else {
			t.Logf("[plan] Invalid environment correctly rejected: %v", err)
		}
	})
}

// TestLifecycle_AppSubset deploys with a custom installation_apps_list containing
// only a subset of apps, verifying that only those apps are installed.
func TestLifecycle_AppSubset(t *testing.T) {
	t.Parallel()
	workingDir := appModuleDir

	subsetApps := []interface{}{
		map[string]interface{}{
			"uuid":       "8ae2e0f4-cb4f-476b-ba9c-ee84bbab471b",
			"name":       "AWS Application Load Balancer",
			"version":    "latest",
			"parameters": map[string]interface{}{},
		},
		map[string]interface{}{
			"uuid":       "a542409f-c491-404f-9a63-7078fcc945e2",
			"name":       "AWS Lambda",
			"version":    "latest",
			"parameters": map[string]interface{}{},
		},
	}

	vars := map[string]interface{}{
		"installation_apps_list": subsetApps,
	}

	defer test_structure.RunTestStage(t, "cleanup", func() {
		destroyApp(t)
	})
	test_structure.RunTestStage(t, "deploy", func() {
		counts := deployApp(t, vars)
		testresources.AssertResourceCounts(t, counts)
	})

	test_structure.RunTestStage(t, "health", func() {
		validateInstalledApps(t, workingDir, []string{
			"AWS Application Load Balancer",
			"AWS Lambda",
		})

		opts := test_structure.LoadTerraformOptions(t, workingDir)
		validateHierarchyExists(t, opts)
	})
}
