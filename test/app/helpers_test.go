package app_test

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SumoLogic/terraform-sumologic-aws-observability/test/testresources"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

const (
	appModuleDir = "../../examples/appmodule"
	appTFVars    = "../../examples/appmodule/main.auto.tfvars"
)

// expectedAppNames is the canonical list of apps installed by the module.
// These are installed into "Installed Apps" — no personal/admin folder.
var expectedAppNames = []string{
	"Amazon Overview",
	"Amazon ECS(Without Container Insights and Traces)",
	"Amazon ECS(With Container Insights and Traces)",
	"Amazon ElastiCache",
	"Amazon RDS",
	"Amazon SNS",
	"Amazon SQS",
	"AWS API Gateway",
	"AWS Application Load Balancer",
	"AWS Classic Load Balancer",
	"AWS DynamoDB",
	"AWS EC2",
	"AWS Lambda",
	"AWS Network Load Balancer",
	"Host Metrics (EC2)",
}

// sharedFieldAddrs are org-level Sumo fields that may already exist and must be
// imported before apply to avoid "already exists" errors.
var sharedFieldAddrs = []string{
	"sumologic_field.account",
	"sumologic_field.region",
	"sumologic_field.accountid",
	"sumologic_field.namespace",
	"sumologic_field.loadbalancer",
	"sumologic_field.loadbalancername",
	"sumologic_field.apiname",
	"sumologic_field.tablename",
	"sumologic_field.instanceid",
	"sumologic_field.clustername",
	"sumologic_field.cacheclusterid",
	"sumologic_field.functionname",
	"sumologic_field.networkloadbalancer",
	"sumologic_field.dbidentifier",
	"sumologic_field.topicname",
	"sumologic_field.queuename",
	"sumologic_field.dbclusteridentifier",
	"sumologic_field.dbinstanceidentifier",
	"sumologic_field.apiid",
}

var appProps map[string]string

func init() {
	appProps = loadProps(appTFVars)
}

func loadProps(path string) map[string]string {
	result := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return result
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx >= 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			val = strings.Trim(val, `"`)
			if key != "" {
				result[key] = val
			}
		}
	}
	return result
}

func getSumologicURL() string {
	raw := appProps["sumo_api_endpoint"]
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func sumoSearchCfg() testresources.Config {
	return testresources.Config{
		SumoBaseURL:   getSumologicURL(),
		SumoAccessID:  appProps["sumologic_access_id"],
		SumoAccessKey: appProps["sumologic_access_key"],
		AWSRegion:     "us-east-1",
	}
}

func runShell(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// importExistingSharedResources imports Sumo fields that already exist org-wide
// to avoid "already exists" apply errors.
func importExistingSharedResources(t *testing.T, opts *terraform.Options) {
	t.Helper()
	cfg := sumoSearchCfg()
	for _, addr := range sharedFieldAddrs {
		parts := strings.SplitN(addr, ".", 2)
		if len(parts) != 2 {
			continue
		}
		fieldName := parts[1]
		fieldID := runShell(fmt.Sprintf(
			`curl -s -u "%s:%s" "%s/api/v1/fields" | jq -r '.data[] | select(.fieldName=="%s") | .fieldId // empty'`,
			cfg.SumoAccessID, cfg.SumoAccessKey, cfg.SumoBaseURL, fieldName,
		))
		if fieldID == "" || fieldID == "null" {
			continue
		}
		_, err := terraform.RunTerraformCommandE(t, opts, "import", "-input=false", addr, fieldID)
		if err != nil {
			t.Logf("[import] %s id=%s: skipped (%v)", addr, fieldID, err)
		} else {
			t.Logf("[import] %s id=%s: imported", addr, fieldID)
		}
	}
}

// releaseSharedResources removes shared fields from state before destroy
// so they are not deleted.
func releaseSharedResources(t *testing.T, opts *terraform.Options) {
	t.Helper()
	for _, addr := range sharedFieldAddrs {
		terraform.RunTerraformCommandE(t, opts, "state", "rm", addr)
	}
	t.Log("[app] Released shared Sumo fields from state")
}

// deployApp runs import → init → apply and saves options.
func deployApp(t *testing.T, vars map[string]interface{}) *terraform.ResourceCount {
	t.Helper()
	opts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: appModuleDir,
		Vars:         vars,
		NoColor:      true,
	})
	test_structure.SaveTerraformOptions(t, appModuleDir, opts)
	terraform.Init(t, opts)
	importExistingSharedResources(t, opts)
	out := terraform.Apply(t, opts)
	return terraform.GetResourceCount(t, out)
}

// redeployApp re-applies with updated vars without reinit.
func redeployApp(t *testing.T, vars map[string]interface{}) *terraform.ResourceCount {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, appModuleDir)
	opts.Vars = vars
	test_structure.SaveTerraformOptions(t, appModuleDir, opts)
	out := terraform.Apply(t, opts)
	return terraform.GetResourceCount(t, out)
}

// destroyApp releases shared fields from state then destroys remaining resources.
func destroyApp(t *testing.T) {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, appModuleDir)
	releaseSharedResources(t, opts)
	_, err := terraform.DestroyE(t, opts)
	if err != nil {
		t.Logf("[destroy] First attempt failed, retrying: %v", err)
		terraform.DestroyE(t, opts)
	}
	cleanStateFiles(appModuleDir)
}

func cleanStateFiles(dir string) {
	os.RemoveAll(filepath.Join(dir, ".test-data"))
	os.Remove(filepath.Join(dir, "terraform.tfstate"))
	os.Remove(filepath.Join(dir, "terraform.tfstate.backup"))
}

// validateInstalledApps checks the installed_apps output contains all expected app names.
func validateInstalledApps(t *testing.T, workingDir string, expectedNames []string) {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	// installed_apps is a map output; use OutputRequired per-key isn't practical,
	// so we read the raw JSON output and check each name appears.
	raw := runShell(fmt.Sprintf(
		`terraform -chdir="%s" output -json installed_apps 2>/dev/null`,
		appModuleDir,
	))
	if raw == "" {
		// Fall back via terraform output command
		raw = terraform.OutputJson(t, opts, "installed_apps")
	}
	passed, failed := 0, 0
	for _, name := range expectedNames {
		if strings.Contains(raw, name) {
			t.Logf("[apps] INSTALLED: %q", name)
			passed++
		} else {
			t.Errorf("[apps] MISSING: app %q not found in installed_apps output", name)
			failed++
		}
	}
	t.Logf("[apps] %d/%d apps validated in Installed Apps", passed, passed+failed)
}

// validateFERExists checks that a FER output ID is non-empty.
func validateFERExists(t *testing.T, opts *terraform.Options, outputKey string) {
	t.Helper()
	val := terraform.Output(t, opts, outputKey)
	if val == "" {
		t.Errorf("[fer] %s: FER ID is empty", outputKey)
	} else {
		t.Logf("[fer] %s: ID=%s (OK)", outputKey, val)
	}
}

// validateFieldExists checks that a Sumo field output ID is non-empty.
func validateFieldExists(t *testing.T, opts *terraform.Options, outputKey string) {
	t.Helper()
	val := terraform.Output(t, opts, outputKey)
	if val == "" {
		t.Errorf("[field] %s: field ID is empty", outputKey)
	} else {
		t.Logf("[field] %s: ID=%s (OK)", outputKey, val)
	}
}

// validateHierarchyExists checks the hierarchy output is non-empty.
func validateHierarchyExists(t *testing.T, opts *terraform.Options) {
	t.Helper()
	val := terraform.Output(t, opts, "hierarchy_id")
	if val == "" {
		t.Error("[hierarchy] hierarchy_id is empty — hierarchy was not created")
	} else {
		t.Logf("[hierarchy] ID=%s (OK)", val)
	}
}

// validateMonitorsFolderExists checks the monitors folder output is non-empty.
func validateMonitorsFolderExists(t *testing.T, opts *terraform.Options) {
	t.Helper()
	val := terraform.Output(t, opts, "monitors_folder_id")
	if val == "" {
		t.Error("[monitors] monitors_folder_id is empty")
	} else {
		t.Logf("[monitors] ID=%s (OK)", val)
	}
}
