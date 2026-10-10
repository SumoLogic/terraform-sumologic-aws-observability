package app_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// sharedFERAddrs are org-level FERs to release from state before destroy (so they are not
// deleted from Sumo). FERs controlled by count use the [0] indexed address; we list both
// the indexed and non-indexed forms so release succeeds regardless of which form is in state.
var sharedFERAddrs = []string{
	// Always count=1 (no conditional) — non-indexed address
	"sumologic_field_extraction_rule.AwsObservabilityElbAccessLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityFieldExtractionRule",
	"sumologic_field_extraction_rule.AwsObservabilityGenericCloudWatchLogsFER",
	// Conditional count — list both non-indexed (legacy) and [0] (current)
	"sumologic_field_extraction_rule.AwsObservabilityRdsCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityRdsCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityCLBCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityCLBCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityNLBCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityNLBCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityEC2CloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityEC2CloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityAlbAccessLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityAlbAccessLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityALBCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityALBCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityApiGatewayCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityApiGatewayCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityApiGatewayAccessLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityApiGatewayAccessLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityDynamoDBCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityDynamoDBCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityECSCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityECSCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityElastiCacheCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityElastiCacheCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilityLambdaCloudWatchLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilityLambdaCloudWatchLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilitySNSCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilitySNSCloudTrailLogsFER[0]",
	"sumologic_field_extraction_rule.AwsObservabilitySQSCloudTrailLogsFER",
	"sumologic_field_extraction_rule.AwsObservabilitySQSCloudTrailLogsFER[0]",
}

// sharedHierarchyAddr is the org-level hierarchy resource address.
const sharedHierarchyAddr = "module.sumo-module.sumologic_hierarchy.awso_hierarchy"

// sharedAppUUIDs maps Terraform resource address → Sumo app UUID for all default apps.
// Used to import existing app instances before apply, and release from state before destroy.
var sharedAppUUIDs = map[string]string{
	"module.sumo-module.sumologic_app.amazon_overview":                                                     "32c8b96c-161c-46d4-b81d-235cc0b56b87",
	`module.sumo-module.sumologic_app.apps["Amazon ECS(Without Container Insights and Traces)"]`:           "c26c3149-6131-4b97-b250-adf53ad7cd16",
	`module.sumo-module.sumologic_app.apps["Amazon ECS(With Container Insights and Traces)"]`:              "69c8db21-9d42-4210-8fa8-365155d90074",
	`module.sumo-module.sumologic_app.apps["Amazon ElastiCache"]`:                                          "1354f831-1745-447e-91b9-0dd6b32eebec",
	`module.sumo-module.sumologic_app.apps["Amazon RDS"]`:                                                  "30519e43-1a92-482c-be08-deaaf09c88b6",
	`module.sumo-module.sumologic_app.apps["Amazon SNS"]`:                                                  "7590a039-5935-47ff-87ea-bc0970e6e96a",
	`module.sumo-module.sumologic_app.apps["Amazon SQS"]`:                                                  "dea50b70-dc1f-406d-884d-3945214861ae",
	`module.sumo-module.sumologic_app.apps["AWS API Gateway"]`:                                             "9b952ef9-7a20-44e4-861a-25f9124846ff",
	`module.sumo-module.sumologic_app.apps["AWS Application Load Balancer"]`:                               "8ae2e0f4-cb4f-476b-ba9c-ee84bbab471b",
	`module.sumo-module.sumologic_app.apps["AWS Classic Load Balancer"]`:                                   "f113016e-4d15-4f4e-a254-4510a995f525",
	`module.sumo-module.sumologic_app.apps["AWS DynamoDB"]`:                                                "7d9b9d0b-0a1f-4af9-9edd-889c2190024b",
	`module.sumo-module.sumologic_app.apps["AWS EC2"]`:                                                     "3dcaacb4-a5de-4e57-a477-fccd04f9e40f",
	`module.sumo-module.sumologic_app.apps["AWS Lambda"]`:                                                  "a542409f-c491-404f-9a63-7078fcc945e2",
	`module.sumo-module.sumologic_app.apps["AWS Network Load Balancer"]`:                                   "3f59d805-e2a8-4e0d-ae90-d40f0eb63671",
	`module.sumo-module.sumologic_app.apps["Host Metrics (EC2)"]`:                                          "5ebe888f-43f0-4021-9225-fd5360e58ca4",
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
			rest := strings.TrimSpace(line[idx+1:])
			if key == "" {
				continue
			}
			var val string
			if strings.HasPrefix(rest, `"`) {
				// Extract quoted value — find the closing quote after the opening one.
				inner := rest[1:]
				if end := strings.Index(inner, `"`); end >= 0 {
					val = inner[:end]
				} else {
					val = inner
				}
			} else {
				// Unquoted: stop at first # comment or whitespace.
				val = strings.Fields(rest)[0]
			}
			result[key] = val
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

// sumoDelete performs an authenticated DELETE to the Sumo Logic API.
func sumoDelete(cfg testresources.Config, path string) error {
	req, err := http.NewRequest("DELETE", cfg.SumoBaseURL+"/api/"+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.SumoAccessID, cfg.SumoAccessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil // already gone — treat as success
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sumoDelete %s: status %d", path, resp.StatusCode)
	}
	return nil
}

// sumoGet performs an authenticated GET to the Sumo Logic API using native Go HTTP.
// Avoids shell escaping issues with credentials that contain special characters.
func sumoGet(cfg testresources.Config, path string) ([]byte, error) {
	req, err := http.NewRequest("GET", cfg.SumoBaseURL+"/api/"+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cfg.SumoAccessID, cfg.SumoAccessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("sumoGet %s: status %d", path, resp.StatusCode)
	}
	return body, nil
}

// importExistingSharedResources imports Sumo org-level resources that may already
// exist to avoid "already exists" apply errors. Covers fields, FERs, hierarchy, and apps.
func importExistingSharedResources(t *testing.T, opts *terraform.Options) {
	t.Helper()
	cfg := sumoSearchCfg()

	// Import fields via /api/v1/fields
	fieldsBody, err := sumoGet(cfg, "v1/fields")
	if err != nil {
		t.Logf("[import] fields lookup failed: %v", err)
	} else {
		var fieldsResp struct {
			Data []struct {
				FieldName string `json:"fieldName"`
				FieldID   string `json:"fieldId"`
			} `json:"data"`
		}
		if json.Unmarshal(fieldsBody, &fieldsResp) == nil {
			fieldByName := make(map[string]string, len(fieldsResp.Data))
			for _, f := range fieldsResp.Data {
				fieldByName[f.FieldName] = f.FieldID
			}
			for _, addr := range sharedFieldAddrs {
				parts := strings.SplitN(addr, ".", 2)
				if len(parts) != 2 {
					continue
				}
				fieldID := fieldByName[parts[1]]
				if fieldID == "" {
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
	}

	// Import FERs via /api/v1/extractionRules
	ferBody, err := sumoGet(cfg, "v1/extractionRules")
	if err != nil {
		t.Logf("[import] FER lookup failed: %v", err)
	} else {
		var ferResp struct {
			Data []struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(ferBody, &ferResp) == nil {
			ferByName := make(map[string]string, len(ferResp.Data))
			for _, f := range ferResp.Data {
				ferByName[f.Name] = f.ID
			}
			for _, addr := range sharedFERAddrs {
				parts := strings.SplitN(addr, ".", 2)
				if len(parts) != 2 {
					continue
				}
				ferID := ferByName[parts[1]]
				if ferID == "" {
					continue
				}
				_, err := terraform.RunTerraformCommandE(t, opts, "import", "-input=false", addr, ferID)
				if err != nil {
					t.Logf("[import] %s id=%s: skipped (%v)", addr, ferID, err)
				} else {
					t.Logf("[import] %s id=%s: imported", addr, ferID)
				}
			}
		}
	}

	// Import hierarchy via /api/v1/entities/hierarchies
	hierBody, err := sumoGet(cfg, "v1/entities/hierarchies")
	if err != nil {
		t.Logf("[import] hierarchy lookup failed: %v", err)
	} else {
		var hierResp struct {
			Data []struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(hierBody, &hierResp) == nil {
			for _, h := range hierResp.Data {
				if h.Name == "AWS Observability" {
					_, err := terraform.RunTerraformCommandE(t, opts, "import", "-input=false", sharedHierarchyAddr, h.ID)
					if err != nil {
						t.Logf("[import] %s id=%s: skipped (%v)", sharedHierarchyAddr, h.ID, err)
					} else {
						t.Logf("[import] %s id=%s: imported", sharedHierarchyAddr, h.ID)
					}
					break
				}
			}
		}
	}

	// Import apps via /api/v2/apps/instances?limit=100
	appsBody, err := sumoGet(cfg, "v2/apps/instances?limit=100")
	if err != nil {
		t.Logf("[import] apps instances lookup failed: %v", err)
	} else {
		var appsResp struct {
			Data []struct {
				UUID string `json:"uuid"`
				ID   string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(appsBody, &appsResp) == nil {
			instanceByUUID := make(map[string]string, len(appsResp.Data))
			for _, a := range appsResp.Data {
				instanceByUUID[a.UUID] = a.ID
			}
			for addr, uuid := range sharedAppUUIDs {
				instanceID := instanceByUUID[uuid]
				if instanceID == "" {
					continue
				}
				_, err := terraform.RunTerraformCommandE(t, opts, "import", "-input=false", addr, instanceID)
				if err != nil {
					t.Logf("[import] %s id=%s: skipped (%v)", addr, instanceID, err)
				} else {
					t.Logf("[import] %s id=%s: imported", addr, instanceID)
				}
			}
			// Patch state: the Sumo API returns the actual version string (e.g. "2.0.1")
			// for older instances, causing TF to detect a diff vs config "latest" and
			// attempt an upgrade that fails with app_already_installed. Setting version
			// to "latest" in state removes the drift so TF skips the upgrade entirely.
			patchAppVersionsInState(t, filepath.Join(opts.TerraformDir, "terraform.tfstate"))
		}
	}
}

// patchAppVersionsInState sets version="latest" for all sumologic_app resources in the TF
// state file, preventing false drift when the API returns the resolved version string instead
// of "latest".
func patchAppVersionsInState(t *testing.T, stateFile string) {
	t.Helper()
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Logf("[import] patchAppVersions: cannot read state (%v)", err)
		return
	}
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Logf("[import] patchAppVersions: cannot parse state (%v)", err)
		return
	}
	resources, _ := state["resources"].([]interface{})
	patched := 0
	for _, r := range resources {
		res, _ := r.(map[string]interface{})
		if res["type"] != "sumologic_app" {
			continue
		}
		for _, inst := range res["instances"].([]interface{}) {
			instance, _ := inst.(map[string]interface{})
			attrs, _ := instance["attributes"].(map[string]interface{})
			if attrs == nil {
				continue
			}
			changed := false
			if v, _ := attrs["version"].(string); v != "latest" {
				attrs["version"] = "latest"
				changed = true
			}
			// Clear parameters so TF sees no diff vs the config's empty parameters map.
			// Old instances store {"version": "1.0.1"} etc. which triggers an Update that
			// re-reads the live version from the API and attempts an unwanted upgrade.
			if p, ok := attrs["parameters"]; ok && p != nil {
				if m, ok := p.(map[string]interface{}); ok && len(m) > 0 {
					attrs["parameters"] = map[string]interface{}{}
					changed = true
				}
			}
			if changed {
				patched++
			}
		}
	}
	if patched == 0 {
		return
	}
	out, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Logf("[import] patchAppVersions: cannot marshal state (%v)", err)
		return
	}
	if err := os.WriteFile(stateFile, out, 0644); err != nil {
		t.Logf("[import] patchAppVersions: cannot write state (%v)", err)
		return
	}
	t.Logf("[import] patchAppVersions: patched %d app version(s) to \"latest\" in state", patched)
}

// sideEffectFERNames lists FERs that app installs create internally as side effects.
// These must be deleted before reinstalling apps that create them, to avoid async-job conflicts.
var sideEffectFERNames = []string{
	"AwsObservabilityALBAccessLogsFER",              // uppercase ALB, created by ALB app (not in field.tf)
	"AwsObservabilityAlbAccessLogsFER",              // lowercase ALB, created by ALB app (field.tf count=0 when ALB installed)
	"AwsObservabilityALBCloudTrailLogsFER",          // created by ALB app (field.tf count=0 when ALB installed)
	"AwsObservabilityLambdaCloudWatchLogsFER",       // created by Lambda app (field.tf count=0 when Lambda installed)
	"AwsObservabilityDynamoDBCloudTrailLogsFER",     // created by DynamoDB app (field.tf count=0 when DynamoDB installed)
	"AwsObservabilityECSCloudTrailLogsFER",          // created by ECS app (field.tf count=0 when ECS installed)
	"AwsObservabilityElastiCacheCloudTrailLogsFER",  // created by ElastiCache app (field.tf count=0 when ElastiCache installed)
	"AwsObservabilitySNSCloudTrailLogsFER",          // created by SNS app (field.tf count=0 when SNS installed)
	"AwsObservabilitySQSCloudTrailLogsFER",          // created by SQS app (field.tf count=0 when SQS installed)
	"AwsObservabilityApiGatewayCloudTrailLogsFER",       // mixed-case, created by field.tf when API GW not installed (field.tf count=0 when API GW installed)
	"AwsObservabilityApiGatewayAccessLogsFER",           // mixed-case, created by field.tf when API GW not installed (field.tf count=0 when API GW installed)
	"AwsObservabilityAPIGatewayAccessLogsFER",           // uppercase, created by API Gateway app as side effect
	"AwsObservabilityAPIGatewayCloudTrailLogsFER",       // uppercase, created by API Gateway app as side effect
	"AwsObservabilityRdsCloudTrailLogsFER",              // mixed-case, created by field.tf when RDS not installed (field.tf count=0 when RDS installed)
	"AwsObservabilityRDSCloudTrailLogsFER",              // uppercase, created by RDS app as side effect
	"AwsObservabilityCLBCloudTrailLogsFER",              // created by CLB app (field.tf count=0 when CLB installed)
	"AwsObservabilityNLBCloudTrailLogsFER",              // created by NLB app (field.tf count=0 when NLB installed)
	"AwsObservabilityEC2CloudTrailLogsFER",              // created by EC2 app (field.tf count=0 when EC2 installed)
}

// sideEffectMetricsRuleNames lists Sumo metrics rules created by app installs as side effects.
var sideEffectMetricsRuleNames = []string{
	"AwsObservabilityAPIGatewayMetricsRule", // created by API Gateway app
}

// cleanAppSideEffectMetricsRules deletes metrics rules that app installs create as side effects.
// The Sumo metrics rules API uses the rule name as the resource ID — no list endpoint exists;
// DELETE /v1/metricsRules/{name} deletes the rule directly (204 success, 404 already gone).
func cleanAppSideEffectMetricsRules(t *testing.T, opts *terraform.Options) {
	t.Helper()
	cfg := sumoSearchCfg()
	for _, name := range sideEffectMetricsRuleNames {
		if err := sumoDelete(cfg, "v1/metricsRules/"+name); err != nil {
			t.Logf("[cleanMetricsRules] delete %s: %v", name, err)
		} else {
			t.Logf("[cleanMetricsRules] deleted %s", name)
		}
	}
}

// ferToAppUUID maps side-effect FER names to the app catalog UUID that creates them.
// Used by cleanAppSideEffectFERs to skip FER deletion when the app is already installed —
// deleting a side-effect FER while its app's install job is still running causes Sumo to
// roll back the partial install, leaving the app out of the instances catalog.
var ferToAppUUID = map[string]string{
	"AwsObservabilityRdsCloudTrailLogsFER": "30519e43-1a92-482c-be08-deaaf09c88b6", // Amazon RDS (mixed-case)
	"AwsObservabilityRDSCloudTrailLogsFER": "30519e43-1a92-482c-be08-deaaf09c88b6", // Amazon RDS (uppercase side-effect)
	"AwsObservabilityALBAccessLogsFER":     "8ae2e0f4-cb4f-476b-ba9c-ee84bbab471b", // ALB
	"AwsObservabilityAlbAccessLogsFER":     "8ae2e0f4-cb4f-476b-ba9c-ee84bbab471b", // ALB
	"AwsObservabilityALBCloudTrailLogsFER": "8ae2e0f4-cb4f-476b-ba9c-ee84bbab471b", // ALB
	"AwsObservabilityLambdaCloudWatchLogsFER":      "a542409f-c491-404f-9a63-7078fcc945e2", // Lambda
	"AwsObservabilityDynamoDBCloudTrailLogsFER":    "7d9b9d0b-0a1f-4af9-9edd-889c2190024b", // DynamoDB
	"AwsObservabilityECSCloudTrailLogsFER":         "c26c3149-6131-4b97-b250-adf53ad7cd16", // ECS (without CI)
	"AwsObservabilityElastiCacheCloudTrailLogsFER": "1354f831-1745-447e-91b9-0dd6b32eebec", // ElastiCache
	"AwsObservabilitySNSCloudTrailLogsFER":         "7590a039-5935-47ff-87ea-bc0970e6e96a", // SNS
	"AwsObservabilitySQSCloudTrailLogsFER":         "dea50b70-dc1f-406d-884d-3945214861ae", // SQS
	"AwsObservabilityApiGatewayCloudTrailLogsFER":  "9b952ef9-7a20-44e4-861a-25f9124846ff", // API Gateway
	"AwsObservabilityApiGatewayAccessLogsFER":      "9b952ef9-7a20-44e4-861a-25f9124846ff", // API Gateway
	"AwsObservabilityAPIGatewayAccessLogsFER":      "9b952ef9-7a20-44e4-861a-25f9124846ff", // API Gateway (uppercase)
	"AwsObservabilityAPIGatewayCloudTrailLogsFER":  "9b952ef9-7a20-44e4-861a-25f9124846ff", // API Gateway (uppercase)
	"AwsObservabilityCLBCloudTrailLogsFER":         "f113016e-4d15-4f4e-a254-4510a995f525", // CLB
	"AwsObservabilityNLBCloudTrailLogsFER":         "3f59d805-e2a8-4e0d-ae90-d40f0eb63671", // NLB
	"AwsObservabilityEC2CloudTrailLogsFER":         "3dcaacb4-a5de-4e57-a477-fccd04f9e40f", // EC2
}

// cleanAppSideEffectFERs deletes FERs that app installs create as side effects.
// Only protects FERs for apps that are BOTH currently installed in Sumo AND in
// the current deployment's subset — those apps' install jobs may be running this
// deploy, so deleting their FERs could trigger a Sumo rollback.
// FERs for apps not in the current subset are always deleted: the module will
// not install those apps this run, so there is no active install job to protect.
func cleanAppSideEffectFERs(t *testing.T, opts *terraform.Options) {
	t.Helper()
	cfg := sumoSearchCfg()

	// Build the set of app UUIDs in the current deployment subset.
	// An empty installation_apps_list means "all apps" — protect all installed.
	subsetUUIDs := make(map[string]bool)
	allApps := true
	if appsList, ok := opts.Vars["installation_apps_list"]; ok {
		if items, ok := appsList.([]interface{}); ok && len(items) > 0 {
			allApps = false
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					if uuid, ok := m["uuid"].(string); ok {
						subsetUUIDs[uuid] = true
					}
				}
			}
		}
	}

	// Fetch currently installed app UUIDs so we can protect their side-effect FERs.
	installedUUIDs := make(map[string]bool)
	if body, err := sumoGet(cfg, "v2/apps/instances?limit=100"); err == nil {
		var appsResp struct {
			Data []struct{ UUID string `json:"uuid"` } `json:"data"`
		}
		if json.Unmarshal(body, &appsResp) == nil {
			for _, a := range appsResp.Data {
				installedUUIDs[a.UUID] = true
			}
		}
	}

	ferBody, err := sumoGet(cfg, "v1/extractionRules")
	if err != nil {
		t.Logf("[cleanFERs] list failed: %v", err)
		return
	}
	var ferResp struct {
		Data []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(ferBody, &ferResp) != nil {
		return
	}
	ferByName := make(map[string]string, len(ferResp.Data))
	for _, f := range ferResp.Data {
		ferByName[f.Name] = f.ID
	}
	for _, name := range sideEffectFERNames {
		id := ferByName[name]
		if id == "" {
			continue
		}
		// Protect a FER only when its owning app is installed AND is being deployed
		// this run (i.e. in the subset, or we're deploying all apps).
		if ownerUUID, ok := ferToAppUUID[name]; ok && installedUUIDs[ownerUUID] {
			if allApps || subsetUUIDs[ownerUUID] {
				t.Logf("[cleanFERs] skipping %s — owning app %s is installed and in current deploy", name, ownerUUID)
				continue
			}
		}
		if err := sumoDelete(cfg, "v1/extractionRules/"+id); err != nil {
			t.Logf("[cleanFERs] delete %s id=%s: %v", name, id, err)
		} else {
			t.Logf("[cleanFERs] deleted %s id=%s", name, id)
		}
		// Remove from state — try both non-indexed (old, no count) and indexed (count=1) addresses
		terraform.RunTerraformCommandE(t, opts, "state", "rm",
			"sumologic_field_extraction_rule."+name)
		terraform.RunTerraformCommandE(t, opts, "state", "rm",
			"sumologic_field_extraction_rule."+name+"[0]")
	}
}

// cleanStaleAppStates removes from TF state any installed apps that are not in the current
// deployment vars. This prevents spurious destroy operations (and their async-job failures)
// during apply. Only acts when installation_apps_list is non-empty (explicit subset).
func cleanStaleAppStates(t *testing.T, opts *terraform.Options) {
	t.Helper()
	appsList, ok := opts.Vars["installation_apps_list"]
	if !ok {
		return
	}
	items, _ := appsList.([]interface{})
	if len(items) == 0 {
		return // empty = use defaults = all apps, nothing to prune
	}
	currentApps := make(map[string]bool, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				currentApps[name] = true
			}
		}
	}
	for addr := range sharedAppUUIDs {
		if !strings.Contains(addr, `sumologic_app.apps["`) {
			continue // skip amazon_overview
		}
		// extract name from: module.sumo-module.sumologic_app.apps["<name>"]
		name := strings.TrimSuffix(strings.SplitN(addr, `apps["`, 2)[1], `"]`)
		if !currentApps[name] {
			terraform.RunTerraformCommandE(t, opts, "state", "rm", addr)
			t.Logf("[cleanStale] removed %s from state (not in current deployment)", addr)
		}
	}
}

// releaseSharedResources removes org-level shared resources from state before destroy
// so they are not deleted from Sumo. Covers fields, FERs, and hierarchy.
// Apps are intentionally NOT released: letting TF destroy them ensures fresh installs
// on the next test run, which re-triggers the app install jobs that create side-effect FERs.
func releaseSharedResources(t *testing.T, opts *terraform.Options) {
	t.Helper()
	for _, addr := range sharedFieldAddrs {
		terraform.RunTerraformCommandE(t, opts, "state", "rm", addr)
	}
	for _, addr := range sharedFERAddrs {
		terraform.RunTerraformCommandE(t, opts, "state", "rm", addr)
	}
	terraform.RunTerraformCommandE(t, opts, "state", "rm", sharedHierarchyAddr)
	t.Log("[app] Released shared Sumo org-level resources from state (fields, FERs, hierarchy)")
}

// pollUntilAppsInstalled polls the Sumo instances API every 30 seconds until all
// sharedAppUUIDs are present in the org (or maxWait elapses). Called after a provider
// timeout so the retry's importExistingSharedResources can pick up newly-completed apps
// instead of triggering a second conflicting install job.
func pollUntilAppsInstalled(t *testing.T, maxWait time.Duration) {
	t.Helper()
	cfg := sumoSearchCfg()
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		time.Sleep(30 * time.Second)
		body, err := sumoGet(cfg, "v2/apps/instances?limit=100")
		if err != nil {
			t.Logf("[pollApps] instances lookup failed: %v", err)
			continue
		}
		var resp struct {
			Data []struct {
				UUID string `json:"uuid"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &resp) != nil {
			continue
		}
		installed := make(map[string]bool, len(resp.Data))
		for _, a := range resp.Data {
			installed[a.UUID] = true
		}
		var missing []string
		for _, uuid := range sharedAppUUIDs {
			if !installed[uuid] {
				missing = append(missing, uuid)
			}
		}
		if len(missing) == 0 {
			t.Logf("[pollApps] all apps present in Sumo")
			return
		}
		t.Logf("[pollApps] %d app(s) not yet installed, waiting...", len(missing))
	}
	t.Logf("[pollApps] max wait elapsed; proceeding with retry (some install jobs may still be running)")
}

func deployApp(t *testing.T, vars map[string]interface{}) *terraform.ResourceCount {
	t.Helper()
	opts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: appModuleDir,
		Vars:         vars,
		NoColor:      true,
	})
	test_structure.SaveTerraformOptions(t, appModuleDir, opts)
	terraform.Init(t, opts)

	// Pre-flight cleanup runs ONCE before the first apply. Handles stale side-effect
	// FERs/metrics-rules left by a previous test run that didn't fully clean up.
	// Must NOT run again between retries — deleting a side-effect FER while an app's
	// install job is still running causes Sumo to roll back the partial install.
	cleanAppSideEffectFERs(t, opts)
	cleanAppSideEffectMetricsRules(t, opts)
	importExistingSharedResources(t, opts)
	cleanStaleAppStates(t, opts)

	// Skip apply-time refresh: preserve the version="latest" patches from importExistingSharedResources
	// so TF doesn't re-read stale version strings from the API and plan a spurious upgrade.
	opts.ExtraArgs.Apply = append(opts.ExtraArgs.Apply, "-refresh=false")

	const maxRetries = 3
	var out string
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			t.Logf("[deployApp] retry %d — re-importing before re-apply", attempt)
			// Only re-import: pick up any apps that completed their install since the
			// last attempt. Do NOT re-clean FERs — ongoing install jobs create FERs as
			// checkpoints; deleting them mid-job triggers a Sumo rollback.
			importExistingSharedResources(t, opts)
		}
		out, lastErr = terraform.ApplyE(t, opts)
		if lastErr == nil {
			break
		}
		errStr := lastErr.Error()
		isTimeout := strings.Contains(errStr, "timeout while waiting for state to become 'Success'")
		// These "already exists" errors occur when the state lock race on macOS causes
		// terraform import to be skipped, leaving shared resources unimported. The retry
		// re-imports them successfully (state file now exists) and the apply finds no diff.
		isAlreadyExists := strings.Contains(errStr, "field:already_exists") ||
			strings.Contains(errStr, "hierarchy:duplicate") ||
			strings.Contains(errStr, "apps:app_already_installed")
		retryable := strings.Contains(errStr, "unexpected end of JSON input") || isTimeout || isAlreadyExists
		if !retryable || attempt >= maxRetries {
			t.Fatalf("[deployApp] apply failed after %d attempt(s): %v", attempt+1, lastErr)
		}
		t.Logf("[deployApp] attempt %d retryable error: %v", attempt+1, lastErr)
		if isTimeout {
			// The provider's 1-minute waitForJob timed out but the Sumo async job is
			// still running. Poll until all apps appear in the instances list (max 15m)
			// so the next retry's import picks them up rather than re-triggering install.
			t.Logf("[deployApp] waiting for Sumo async install jobs to complete (max 15m)...")
			pollUntilAppsInstalled(t, 15*time.Minute)
		}
	}

	// Restore opts without -refresh=false for subsequent operations (redeployApp, destroy).
	opts.ExtraArgs.Apply = nil
	test_structure.SaveTerraformOptions(t, appModuleDir, opts)
	return terraform.GetResourceCount(t, out)
}

// redeployApp re-applies with updated vars without reinit.
// Releases shared fields from state and re-imports them with current Sumo IDs before
// applying. The Sumo provider may return 404 for field IDs assigned by the previous
// apply (eventual consistency), causing false "deleted externally" drift that triggers
// an "already exists" error on the subsequent create attempt.
func redeployApp(t *testing.T, vars map[string]interface{}) *terraform.ResourceCount {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, appModuleDir)
	opts.Vars = vars
	opts.ExtraArgs.Apply = append(opts.ExtraArgs.Apply, "-refresh=false")
	test_structure.SaveTerraformOptions(t, appModuleDir, opts)
	for _, addr := range sharedFieldAddrs {
		terraform.RunTerraformCommandE(t, opts, "state", "rm", addr)
	}
	importExistingSharedResources(t, opts)
	out := terraform.Apply(t, opts)
	opts.ExtraArgs.Apply = nil
	test_structure.SaveTerraformOptions(t, appModuleDir, opts)
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

// appNameToUUID maps display name → Sumo app UUID, derived from sharedAppUUIDs.
// Used to verify app installation via the Sumo API for apps not in the installed_apps output.
var appNameToUUID = map[string]string{}

func init() {
	for addr, uuid := range sharedAppUUIDs {
		if strings.Contains(addr, `sumologic_app.apps["`) {
			name := strings.TrimSuffix(strings.SplitN(addr, `apps["`, 2)[1], `"]`)
			appNameToUUID[name] = uuid
		} else if strings.HasSuffix(addr, "sumologic_app.amazon_overview") {
			appNameToUUID["Amazon Overview"] = uuid
		}
	}
}

// validateInstalledApps checks all expected app names are installed.
// Apps found in the installed_apps TF output are checked there; any not found
// (e.g. "Amazon Overview", a separate static resource excluded from the map output)
// are verified via the Sumo API using their known UUIDs.
func validateInstalledApps(t *testing.T, workingDir string, expectedNames []string) {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	raw := runShell(fmt.Sprintf(
		`terraform -chdir="%s" output -json installed_apps 2>/dev/null`,
		appModuleDir,
	))
	if raw == "" {
		raw = terraform.OutputJson(t, opts, "installed_apps")
	}

	// Build UUID → instanceID map from Sumo for apps that need API fallback.
	installedUUIDs := make(map[string]bool)
	cfg := sumoSearchCfg()
	if body, err := sumoGet(cfg, "v2/apps/instances?limit=100"); err == nil {
		var resp struct {
			Data []struct {
				UUID string `json:"uuid"`
				ID   string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &resp) == nil {
			for _, a := range resp.Data {
				installedUUIDs[a.UUID] = true
			}
		}
	}

	passed, failed := 0, 0
	for _, name := range expectedNames {
		if strings.Contains(raw, name) {
			t.Logf("[apps] INSTALLED (output): %q", name)
			passed++
			continue
		}
		// Not in TF output — verify via Sumo API by UUID (covers amazon_overview).
		uuid := appNameToUUID[name]
		if uuid != "" && installedUUIDs[uuid] {
			t.Logf("[apps] INSTALLED (sumo api): %q uuid=%s", name, uuid)
			passed++
		} else {
			t.Errorf("[apps] MISSING: %q not in installed_apps output or Sumo API (uuid=%s)", name, uuid)
			failed++
		}
	}
	t.Logf("[apps] %d/%d apps validated in Installed Apps", passed, passed+failed)
}

// outputKeyToFERName maps TF output keys (for app-managed FERs with count=0 in field.tf)
// to their Sumo FER names, used as a fallback when the TF output is empty.
var outputKeyToFERName = map[string]string{
	"sumologic_field_extraction_rule_alb":         "AwsObservabilityALBAccessLogsFER",
	"sumologic_field_extraction_rule_alb_cloudtrail": "AwsObservabilityALBCloudTrailLogsFER",
	"sumologic_field_extraction_rule_lambda_cw":   "AwsObservabilityLambdaCloudWatchLogsFER",
	"sumologic_field_extraction_rule_dynamodb":    "AwsObservabilityDynamoDBCloudTrailLogsFER",
	"sumologic_field_extraction_rule_ecs":         "AwsObservabilityECSCloudTrailLogsFER",
	"sumologic_field_extraction_rule_elasticache": "AwsObservabilityElastiCacheCloudTrailLogsFER",
	"sumologic_field_extraction_rule_sns":         "AwsObservabilitySNSCloudTrailLogsFER",
	"sumologic_field_extraction_rule_sqs":                       "AwsObservabilitySQSCloudTrailLogsFER",
	"sumologic_field_extraction_rule_apigateway":                "AwsObservabilityAPIGatewayCloudTrailLogsFER",
	"sumologic_field_extraction_rule_apigateway_access_logs":    "AwsObservabilityAPIGatewayAccessLogsFER",
	"sumologic_field_extraction_rule_rds":                       "AwsObservabilityRDSCloudTrailLogsFER",
	"sumologic_field_extraction_rule_clb_cloudtrail":            "AwsObservabilityCLBCloudTrailLogsFER",
	"sumologic_field_extraction_rule_nlb_cloudtrail":            "AwsObservabilityNLBCloudTrailLogsFER",
	"sumologic_field_extraction_rule_ec2metrics":                "AwsObservabilityEC2CloudTrailLogsFER",
}

// validateFERExists checks that a FER output ID is non-empty.
// When the output is empty (app-managed FER with count=0 in field.tf), falls back to
// verifying FER existence via Sumo API by name.
func validateFERExists(t *testing.T, opts *terraform.Options, outputKey string) {
	t.Helper()
	val := terraform.Output(t, opts, outputKey)
	if val != "" {
		t.Logf("[fer] %s: ID=%s (OK)", outputKey, val)
		return
	}
	// Output empty — FER may be app-managed (count=0). Verify via Sumo API.
	ferName, ok := outputKeyToFERName[outputKey]
	if !ok {
		t.Errorf("[fer] %s: FER ID is empty", outputKey)
		return
	}
	cfg := sumoSearchCfg()
	ferBody, err := sumoGet(cfg, "v1/extractionRules")
	if err != nil {
		t.Errorf("[fer] %s: cannot list FERs from Sumo: %v", outputKey, err)
		return
	}
	var ferResp struct {
		Data []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ferBody, &ferResp); err != nil {
		t.Errorf("[fer] %s: cannot parse FER list: %v", outputKey, err)
		return
	}
	for _, f := range ferResp.Data {
		if f.Name == ferName {
			t.Logf("[fer] %s: '%s' exists in Sumo (ID=%s, app-managed)", outputKey, ferName, f.ID)
			return
		}
	}
	t.Errorf("[fer] %s: FER ID is empty AND '%s' not found in Sumo", outputKey, ferName)
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
