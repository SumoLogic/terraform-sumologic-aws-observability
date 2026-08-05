package testresources

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
)

// DeployTerraform runs init → apply and saves TerraformOptions for later stages.
func DeployTerraform(t *testing.T, workingDir string, vars map[string]interface{}, varFiles []string) *terraform.ResourceCount {
	opts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: workingDir,
		Vars:         vars,
		VarFiles:     varFiles,
		NoColor:      true,
	})
	test_structure.SaveTerraformOptions(t, workingDir, opts)
	terraform.Init(t, opts)
	out := terraform.Apply(t, opts)
	return terraform.GetResourceCount(t, out)
}

// RedeployTerraform updates saved options with new vars and re-applies (for update tests).
func RedeployTerraform(t *testing.T, workingDir string, vars map[string]interface{}, varFiles []string) *terraform.ResourceCount {
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	opts.Vars = vars
	opts.VarFiles = varFiles
	test_structure.SaveTerraformOptions(t, workingDir, opts)
	out := terraform.Apply(t, opts)
	return terraform.GetResourceCount(t, out)
}

// DestroyTerraform destroys resources and cleans up state files.
func DestroyTerraform(t *testing.T, workingDir string) {
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	_, err := terraform.DestroyE(t, opts)
	if err != nil {
		t.Logf("[destroy] First attempt failed, retrying: %v", err)
		terraform.DestroyE(t, opts)
	}
	cleanStateFiles(workingDir)
}

func cleanStateFiles(workingDir string) {
	os.RemoveAll(filepath.Join(workingDir, ".test-data"))
	os.Remove(filepath.Join(workingDir, "terraform.tfstate"))
	os.Remove(filepath.Join(workingDir, "terraform.tfstate.backup"))
	exec.Command("sh", "-c", fmt.Sprintf("rm -f %s/terraform.tfstate.*.backup", workingDir)).Run()
}

// AssertResourceCounts logs add/change/destroy counts and fails if any unexpected destroys occurred.
func AssertResourceCounts(t *testing.T, actual *terraform.ResourceCount) {
	t.Helper()
	t.Logf("Resources: %d added, %d changed, %d destroyed", actual.Add, actual.Change, actual.Destroy)
	assert.Greater(t, actual.Add, 0, "Expected at least one resource to be created")
	assert.Equal(t, 0, actual.Destroy, "Unexpected resource destroys during deploy")
}

// AssertResourceExistence validates that expected TF resource addresses appear in state.
// Missing = ERROR, extra = WARNING only (mirrors CF ResourceExistence assertion behaviour).
func AssertResourceExistence(t *testing.T, workingDir string, expectedResources []string) {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	stateOutput := terraform.RunTerraformCommand(t, opts, "state", "list")
	actualList := splitLines(stateOutput)

	actualSet := make(map[string]bool, len(actualList))
	for _, r := range actualList {
		actualSet[r] = true
	}

	missing := 0
	for _, r := range expectedResources {
		if !actualSet[r] {
			t.Errorf("MISSING resource in state: %s", r)
			missing++
		}
	}

	expectedSet := make(map[string]bool, len(expectedResources))
	for _, r := range expectedResources {
		expectedSet[r] = true
	}
	for _, r := range actualList {
		if !expectedSet[r] {
			t.Logf("INFO: extra resource in state (not in expected list): %s", r)
		}
	}

	t.Logf("ResourceExistence: %d expected, %d actual, %d missing",
		len(expectedResources), len(actualList), missing)
}

// AssertTagsAbsent fails if any of the given tag keys appear in the actual tag map.
func AssertTagsAbsent(t *testing.T, resourceLabel string, actualTags map[string]string, removedKeys []string) {
	t.Helper()
	for _, key := range removedKeys {
		if _, found := actualTags[key]; found {
			t.Errorf("[tags] %s still has removed tag key %q", resourceLabel, key)
		} else {
			t.Logf("[tags] %s: removed tag key %q is absent (OK)", resourceLabel, key)
		}
	}
}

func splitLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
