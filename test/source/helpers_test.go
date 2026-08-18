package source_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
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
	testSourceDir    = "../../examples/sourcemodule/testSource"
	overrideDir      = "../../examples/sourcemodule/overrideSources"
	testSourceTFVars = "../../examples/sourcemodule/testSource/main.auto.tfvars"
	region           = "us-east-1"
)

// e2eRetryDelays matches the CF test retry schedule for Sumo source validation.
var e2eRetryDelays = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	2 * time.Minute,
	2 * time.Minute,
	2 * time.Minute,
}

var (
	props map[string]string
)

func init() {
	props = loadProps(testSourceTFVars)
}

func loadProps(path string) map[string]string {
	result := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[helpers] Warning: could not open %s: %v", path, err)
		}
		return result
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx >= 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			// Handle quoted values with inline comments: "value" # comment
			if strings.HasPrefix(val, `"`) {
				if end := strings.Index(val[1:], `"`); end >= 0 {
					val = val[1 : end+1]
				} else {
					val = strings.Trim(val, `"`)
				}
			} else {
				// Unquoted value — strip inline comment
				if hashIdx := strings.Index(val, " #"); hashIdx >= 0 {
					val = strings.TrimSpace(val[:hashIdx])
				}
			}
			if key != "" {
				result[key] = val
			}
		}
	}
	return result
}

func getProperty(key string) string {
	return props[key]
}

func getSumologicURL() string {
	raw := getProperty("sumo_api_endpoint")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func sumoSearchCfg() testresources.Config {
	return testresources.Config{
		SumoBaseURL:   getSumologicURL(),
		SumoAccessID:  getProperty("sumologic_access_id"),
		SumoAccessKey: getProperty("sumologic_access_key"),
		AWSRegion:     region,
	}
}

func awsCfg() testresources.Config {
	return testresources.Config{AWSRegion: region}
}

// writeVarsFile writes optional HCL content to a temporary vars file in workingDir.
// Returns the absolute path. Pass nil content for an empty file.
func writeVarsFile(t *testing.T, workingDir string, content []byte) string {
	t.Helper()
	abs, err := filepath.Abs(workingDir)
	if err != nil {
		t.Fatalf("[helpers] filepath.Abs(%s): %v", workingDir, err)
	}
	path := filepath.Join(abs, "vars.tfvars")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("[helpers] writeVarsFile: %v", err)
	}
	return path
}

// ferToTFResource maps lowercased Sumo Logic FER names to their Terraform resource addresses in field.tf.
// Keys are lowercase so lookups are case-insensitive (the API and Terraform sometimes differ in ALB vs Alb etc.).
var ferToTFResource = map[string]string{
	"awsobservabilityalbaccesslogsfer":              "sumologic_field_extraction_rule.AwsObservabilityAlbAccessLogsFER",
	"awsobservabilityapigatewaycloudtraillogsfer":   "sumologic_field_extraction_rule.AwsObservabilityApiGatewayCloudTrailLogsFER",
	"awsobservabilitydynamodbcloudtraillogsfer":     "sumologic_field_extraction_rule.AwsObservabilityDynamoDBCloudTrailLogsFER",
	"awsobservabilityec2cloudtraillogsfer":          "sumologic_field_extraction_rule.AwsObservabilityEC2CloudTrailLogsFER",
	"awsobservabilityecscloudtraillogsfer":          "sumologic_field_extraction_rule.AwsObservabilityECSCloudTrailLogsFER",
	"awsobservabilityelasticachecloudtraillogsfer":  "sumologic_field_extraction_rule.AwsObservabilityElastiCacheCloudTrailLogsFER",
	"awsobservabilityelbaccesslogsfer":              "sumologic_field_extraction_rule.AwsObservabilityElbAccessLogsFER",
	"awsobservabilityfieldextractionrule":           "sumologic_field_extraction_rule.AwsObservabilityFieldExtractionRule",
	"awsobservabilitylambdacloudwatchlogsfer":       "sumologic_field_extraction_rule.AwsObservabilityLambdaCloudWatchLogsFER",
	"awsobservabilitygenericcloudwatchlogsfer":      "sumologic_field_extraction_rule.AwsObservabilityGenericCloudWatchLogsFER",
	"awsobservabilityrdscloudtraillogsfer":          "sumologic_field_extraction_rule.AwsObservabilityRdsCloudTrailLogsFER",
	"awsobservabilitysnscloudtraillogsfer":          "sumologic_field_extraction_rule.AwsObservabilitySNSCloudTrailLogsFER",
}

// fieldToTFResource maps Sumo Logic field names to their Terraform resource addresses in field.tf.
var fieldToTFResource = map[string]string{
	"account":              "sumologic_field.account",
	"region":               "sumologic_field.region",
	"accountid":            "sumologic_field.accountid",
	"namespace":            "sumologic_field.namespace",
	"loadbalancer":         "sumologic_field.loadbalancer",
	"loadbalancername":     "sumologic_field.loadbalancername",
	"apiname":              "sumologic_field.apiname",
	"tablename":            "sumologic_field.tablename",
	"instanceid":           "sumologic_field.instanceid",
	"clustername":          "sumologic_field.clustername",
	"cacheclusterid":       "sumologic_field.cacheclusterid",
	"functionname":         "sumologic_field.functionname",
	"networkloadbalancer":  "sumologic_field.networkloadbalancer",
	"dbidentifier":         "sumologic_field.dbidentifier",
	"dbclusteridentifier":  "sumologic_field.dbclusteridentifier",
	"dbinstanceidentifier": "sumologic_field.dbinstanceidentifier",
	"topicname":            "sumologic_field.topicname",
}

// importExistingSumoFields queries the Sumo Logic Fields API and imports pre-existing fields
// into Terraform state. This prevents field:already_exists errors when fields were left
// behind by a prior partial deployment whose state was cleaned up.
//
// Uses exec.Command directly with -lock=false to bypass stale lock files that may remain
// from interrupted previous runs.
func importExistingSumoFields(t *testing.T, opts *terraform.Options) {
	t.Helper()
	baseURL := getSumologicURL()
	if baseURL == "" {
		t.Log("[import] no Sumo API endpoint configured, skipping field import")
		return
	}

	existing := listSumoFields(t, baseURL)
	if len(existing) == 0 {
		t.Log("[import] no fields returned from Sumo API")
		return
	}

	// Resolve absolute path to avoid relative-path issues in child processes.
	absDir, err := filepath.Abs(opts.TerraformDir)
	if err != nil {
		t.Logf("[import] could not resolve terraform dir %s: %v", opts.TerraformDir, err)
		return
	}

	// Remove stale lock file if present (left by killed previous run).
	lockFile := filepath.Join(absDir, ".terraform.tfstate.lock.info")
	if _, statErr := os.Stat(lockFile); statErr == nil {
		t.Logf("[import] removing stale state lock file %s", lockFile)
		os.Remove(lockFile)
	}

	// Check what resources are already in state so we can skip them.
	stateCmd := exec.Command("terraform", "state", "list")
	stateCmd.Dir = absDir
	stateOut, _ := stateCmd.Output()
	inState := make(map[string]bool)
	for _, line := range strings.Split(string(stateOut), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			inState[trimmed] = true
		}
	}

	for name, id := range existing {
		addr, ok := fieldToTFResource[strings.ToLower(name)]
		if !ok {
			continue
		}
		if inState[addr] {
			t.Logf("[import] %s already in state, skipping", addr)
			continue
		}
		// Use exec.Command directly with -lock=false to avoid any stale-lock failures.
		cmd := exec.Command("terraform", "import", "-lock=false", addr, id)
		cmd.Dir = absDir
		out, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			t.Logf("[import] FAILED %s (id=%s): %v\nOutput: %s", addr, id, cmdErr, string(out))
		} else {
			t.Logf("[import] imported existing Sumo field %q → %s", name, addr)
		}
	}

	// Import pre-existing Field Extraction Rules to avoid fer:invalid_extraction_rule on re-runs.
	existingFERs := listSumoFERs(t, baseURL)
	for name, id := range existingFERs {
		addr, ok := ferToTFResource[name]
		if !ok {
			continue
		}
		if inState[addr] {
			t.Logf("[import] %s already in state, skipping", addr)
			continue
		}
		cmd := exec.Command("terraform", "import", "-lock=false", addr, id)
		cmd.Dir = absDir
		out, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			t.Logf("[import] FAILED FER %s (id=%s): %v\nOutput: %s", addr, id, cmdErr, string(out))
		} else {
			t.Logf("[import] imported existing FER %q → %s", name, addr)
		}
	}
}

// listSumoFields returns a name→id map of all fields in the Sumo Logic org.
func listSumoFields(t *testing.T, baseURL string) map[string]string {
	t.Helper()
	accessID := getProperty("sumologic_access_id")
	accessKey := getProperty("sumologic_access_key")
	if accessID == "" || accessKey == "" {
		return nil
	}
	apiURL := baseURL + "/api/v1/fields"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		t.Logf("[import] build request failed: %v", err)
		return nil
	}
	req.SetBasicAuth(accessID, accessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("[import] GET %s failed: %v", apiURL, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Logf("[import] GET %s returned %d", apiURL, resp.StatusCode)
		return nil
	}
	var payload struct {
		Data []struct {
			FieldID   string `json:"fieldId"`
			FieldName string `json:"fieldName"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Logf("[import] parse fields response: %v", err)
		return nil
	}
	result := make(map[string]string, len(payload.Data))
	for _, f := range payload.Data {
		result[strings.ToLower(f.FieldName)] = f.FieldID
	}
	return result
}

// listSumoFERs returns a name→id map of all Field Extraction Rules in the Sumo Logic org.
func listSumoFERs(t *testing.T, baseURL string) map[string]string {
	t.Helper()
	accessID := getProperty("sumologic_access_id")
	accessKey := getProperty("sumologic_access_key")
	if accessID == "" || accessKey == "" {
		return nil
	}
	apiURL := baseURL + "/api/v1/extractionRules"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		t.Logf("[import] build FER request failed: %v", err)
		return nil
	}
	req.SetBasicAuth(accessID, accessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("[import] GET %s failed: %v", apiURL, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Logf("[import] GET %s returned %d", apiURL, resp.StatusCode)
		return nil
	}
	var payload struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Logf("[import] parse FERs response: %v", err)
		return nil
	}
	result := make(map[string]string, len(payload.Data))
	for _, fer := range payload.Data {
		result[strings.ToLower(fer.Name)] = fer.ID
	}
	return result
}

// importExistingCollector queries the Sumo Logic Collectors API and imports a pre-existing
// collector into Terraform state to avoid "collectors.validation.name.duplicate" errors.
func importExistingCollector(t *testing.T, opts *terraform.Options) {
	t.Helper()
	baseURL := getSumologicURL()
	if baseURL == "" {
		return
	}
	accessID := getProperty("sumologic_access_id")
	accessKey := getProperty("sumologic_access_key")
	if accessID == "" || accessKey == "" {
		return
	}

	absDir, err := filepath.Abs(opts.TerraformDir)
	if err != nil {
		return
	}

	addr := `module.collection-module.sumologic_collector.collector["collector"]`

	// Check if already in state
	stateCmd := exec.Command("terraform", "state", "list")
	stateCmd.Dir = absDir
	stateOut, _ := stateCmd.Output()
	for _, line := range strings.Split(string(stateOut), "\n") {
		if strings.TrimSpace(line) == addr {
			t.Logf("[import] collector already in state, skipping")
			return
		}
	}

	// Build expected collector name: "AWS Observability <alias> <account_id>"
	alias := getProperty("aws_account_alias")

	// Paginate through all collectors (org may have 1000+)
	offset := 0
	limit := 1000
	for {
		apiURL := fmt.Sprintf("%s/api/v1/collectors?limit=%d&offset=%d", baseURL, limit, offset)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return
		}
		req.SetBasicAuth(accessID, accessKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Logf("[import] GET collectors failed: %v", err)
			return
		}

		var payload struct {
			Collectors []struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"collectors"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			t.Logf("[import] parse collectors response: %v", err)
			return
		}
		resp.Body.Close()

		if len(payload.Collectors) == 0 {
			break
		}

		for _, c := range payload.Collectors {
			if alias != "" && strings.Contains(c.Name, alias) {
				id := fmt.Sprintf("%d", c.ID)
				cmd := exec.Command("terraform", "import", "-lock=false", addr, id)
				cmd.Dir = absDir
				out, cmdErr := cmd.CombinedOutput()
				if cmdErr != nil {
					t.Logf("[import] FAILED collector (id=%s): %v\n%s", id, cmdErr, string(out))
				} else {
					t.Logf("[import] imported existing collector %q (id=%s) → %s", c.Name, id, addr)
				}
				return
			}
		}

		if len(payload.Collectors) < limit {
			break
		}
		offset += limit
	}
	t.Log("[import] no matching collector found in Sumo org")
}

// deployTerraform runs init → import existing fields/collector → apply, saves opts and returns resource counts.
func deployTerraform(t *testing.T, workingDir string, vars map[string]interface{}, varsFile string) *terraform.ResourceCount {
	t.Helper()
	varFiles := []string{}
	if varsFile != "" {
		varFiles = []string{varsFile}
	}
	opts := testresources.InitTerraform(t, workingDir, vars, varFiles)
	importExistingSumoFields(t, opts)
	importExistingCollector(t, opts)
	return testresources.ApplyTerraform(t, opts)
}

// redeployTerraform applies updated vars without re-init (for update tests).
func redeployTerraform(t *testing.T, workingDir string, vars map[string]interface{}, varsFile string) *terraform.ResourceCount {
	t.Helper()
	varFiles := []string{}
	if varsFile != "" {
		varFiles = []string{varsFile}
	}
	return testresources.RedeployTerraform(t, workingDir, vars, varFiles)
}

// removeSumoFieldsFromState removes ALL sumologic_field resources from state before destroy.
// Fields persist in the Sumo org (they're org-wide and referenced by sources/FERs across
// many deployments). Removing them from state prevents terraform destroy from attempting to
// delete them — which would fail anyway because Sumo rejects deletion of in-use fields.
func removeSumoFieldsFromState(t *testing.T, opts *terraform.Options) {
	t.Helper()

	absDir, err := filepath.Abs(opts.TerraformDir)
	if err != nil {
		t.Logf("[cleanup] could not resolve terraform dir: %v", err)
		return
	}

	// Remove stale lock file if present.
	lockFile := filepath.Join(absDir, ".terraform.tfstate.lock.info")
	if _, statErr := os.Stat(lockFile); statErr == nil {
		os.Remove(lockFile)
	}

	stateCmd := exec.Command("terraform", "state", "list")
	stateCmd.Dir = absDir
	stateOut, err := stateCmd.Output()
	if err != nil || len(stateOut) == 0 {
		return
	}

	for _, addr := range strings.Split(string(stateOut), "\n") {
		addr = strings.TrimSpace(addr)
		if !strings.HasPrefix(addr, "sumologic_field.") {
			continue
		}
		rmCmd := exec.Command("terraform", "state", "rm", "-lock=false", addr)
		rmCmd.Dir = absDir
		if out, rmErr := rmCmd.CombinedOutput(); rmErr != nil {
			t.Logf("[cleanup] could not remove %s from state: %v\n%s", addr, rmErr, string(out))
		} else {
			t.Logf("[cleanup] removed %s from state (field persists in Sumo org)", addr)
		}
	}
}

// destroyTerraformAllowingErrors runs terraform destroy and returns any error rather than
// failing the test. Use this when the destroy is expected to fail (e.g. BucketNotEmpty).
func destroyTerraformAllowingErrors(t *testing.T, workingDir string) error {
	t.Helper()
	abs, _ := filepath.Abs(workingDir)
	optsFile := filepath.Join(abs, ".test-data", "TerraformOptions.json")
	if _, statErr := os.Stat(optsFile); statErr != nil {
		return nil
	}
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	removeSumoFieldsFromState(t, opts)
	opts.ExtraArgs.Destroy = append(opts.ExtraArgs.Destroy, "-lock=false")
	test_structure.SaveTerraformOptions(t, workingDir, opts)

	_, err := terraform.DestroyE(t, opts)
	// Clean up state files regardless of destroy outcome.
	os.RemoveAll(filepath.Join(abs, ".test-data"))
	os.Remove(filepath.Join(abs, "terraform.tfstate"))
	os.Remove(filepath.Join(abs, "terraform.tfstate.backup"))
	return err
}

// destroyTerraform tears down and cleans state files.
func destroyTerraform(t *testing.T, workingDir string) {
	t.Helper()
	abs, _ := filepath.Abs(workingDir)
	optsFile := filepath.Join(abs, ".test-data", "TerraformOptions.json")
	if _, statErr := os.Stat(optsFile); statErr == nil {
		opts := test_structure.LoadTerraformOptions(t, workingDir)
		removeSumoFieldsFromState(t, opts)
		// Pass -lock=false on destroy to tolerate stale lock files from interrupted runs.
		opts.ExtraArgs.Destroy = append(opts.ExtraArgs.Destroy, "-lock=false")
		test_structure.SaveTerraformOptions(t, workingDir, opts)
	}
	testresources.DestroyTerraform(t, workingDir)
}

// runShell executes a shell command, returns trimmed stdout. Empty string on error.
func runShell(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// validateALBAccessLogsEnabled polls ALB attributes every 30s up to 120s (CF-style).
func validateALBAccessLogsEnabled(t *testing.T, albARN string) {
	t.Helper()
	deadline := time.Now().Add(120 * time.Second)
	for {
		out := runShell(fmt.Sprintf(
			`aws elbv2 describe-load-balancer-attributes --load-balancer-arn "%s" --region %s --query 'Attributes[?Key==`+"`access_logs.s3.enabled`"+`].Value' --output text`,
			albARN, region,
		))
		if strings.TrimSpace(out) == "true" {
			t.Logf("[e2e] ALB %s: access_logs.s3.enabled=true", albARN)
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("[e2e] ALB %s: access_logs.s3.enabled=%q after 120s, want true", albARN, out)
			return
		}
		time.Sleep(30 * time.Second)
	}
}

// validateCLBAccessLogsEnabled polls CLB attributes every 30s up to 120s.
func validateCLBAccessLogsEnabled(t *testing.T, clbName string) {
	t.Helper()
	deadline := time.Now().Add(120 * time.Second)
	for {
		out := runShell(fmt.Sprintf(
			`aws elb describe-load-balancer-attributes --load-balancer-name "%s" --region %s --query 'LoadBalancerAttributes.AccessLog.Enabled' --output text`,
			clbName, region,
		))
		v := strings.ToLower(strings.TrimSpace(out))
		if v == "true" {
			t.Logf("[e2e] CLB %s: AccessLog.Enabled=true", clbName)
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("[e2e] CLB %s: AccessLog.Enabled=%q after 120s, want true", clbName, out)
			return
		}
		time.Sleep(30 * time.Second)
	}
}

// validateSubscriptionFilterExists polls CW Logs for a subscription filter pointing to kfStreamARN.
func validateSubscriptionFilterExists(t *testing.T, logGroupName, kfStreamARN string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		out := runShell(fmt.Sprintf(
			`aws logs describe-subscription-filters --log-group-name "%s" --region %s --query 'subscriptionFilters[?destinationArn==`+"`%s`"+`] | length(@)' --output text`,
			logGroupName, region, kfStreamARN,
		))
		if strings.TrimSpace(out) != "" && strings.TrimSpace(out) != "0" {
			t.Logf("[e2e] Subscription filter found for %s → %s", logGroupName, kfStreamARN)
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("[e2e] No subscription filter on %s pointing to %s after 90s", logGroupName, kfStreamARN)
			return
		}
		time.Sleep(15 * time.Second)
	}
}

// E2EConfig holds pre-created load balancer details for end-to-end validation.
type E2EConfig struct {
	ALBARN  string
	ALBDNS  string
	CLBName string
	CLBDNS  string
}

// runStandardE2E is the single entry point for E2E validation that every source test uses.
// Mirrors CF's E2E phase: verify access-log attributes → generate traffic → wait → validate Sumo.
func runStandardE2E(t *testing.T, vars map[string]interface{}, workingDir string, e2e E2EConfig) {
	t.Helper()

	// Step 1: Verify access logging enabled (CF polls 30s, 120s timeout)
	if getBoolVar(vars, "collect_elb", true) && e2e.ALBARN != "" {
		validateALBAccessLogsEnabled(t, e2e.ALBARN)
	}
	if getBoolVar(vars, "collect_classic_lb", true) && e2e.CLBName != "" {
		validateCLBAccessLogsEnabled(t, e2e.CLBName)
	}

	// Step 2: Generate traffic for each enabled source type
	generateTraffic(t, vars, e2e)

	// Step 3: Wait for S3-based delivery pipelines (5-min delivery + buffer)
	if needsPipelineWait(vars) {
		t.Log("[e2e] Waiting 6 minutes for S3-based delivery pipelines...")
		time.Sleep(6 * time.Minute)
	}

	// Step 4: Validate Sumo data flow per source
	validateSumoDataFlow(t, vars, workingDir)
}

func generateTraffic(t *testing.T, vars map[string]interface{}, e2e E2EConfig) {
	if getBoolVar(vars, "collect_elb", true) && e2e.ALBDNS != "" {
		testresources.GenerateALBTraffic(t, e2e.ALBDNS, 30)
	}
	if getBoolVar(vars, "collect_classic_lb", true) && e2e.CLBDNS != "" {
		testresources.GenerateCLBTraffic(t, e2e.CLBDNS, 20)
	}
	if getBoolVar(vars, "collect_cloudtrail", true) {
		testresources.GenerateCloudTrailTraffic(t, region)
	}
	switch getStrVar(vars, "collect_logs_cloudwatch", "Kinesis Firehose Log Source") {
	case "Kinesis Firehose Log Source", "Lambda Log Forwarder":
		testresources.GenerateCWLogsTraffic(t, region, "/awso/e2e/test", 20)
	}
}

func needsPipelineWait(vars map[string]interface{}) bool {
	return getBoolVar(vars, "collect_elb", true) ||
		getBoolVar(vars, "collect_classic_lb", true) ||
		getBoolVar(vars, "collect_cloudtrail", true)
}

// sourceOutputKeys maps variable flags to their Terraform output keys for source IDs.
var sourceOutputKeys = []struct {
	label    string
	varKey   string
	matchFn  func(interface{}) bool
	outputKey string
}{
	{"elb", "collect_elb", isBoolTrue, "sumologic_elb_source"},
	{"clb", "collect_classic_lb", isBoolTrue, "sumologic_classic_lb_source"},
	{"cloudtrail", "collect_cloudtrail", isBoolTrue, "sumologic_cloudtrail_source"},
	{"kf_logs", "collect_logs_cloudwatch", isKFLogs, "sumologic_kinesis_firehose_for_logs_source"},
	{"kf_metrics", "collect_metric_cloudwatch", isKFMetrics, "sumologic_kinesis_firehose_for_metrics_source"},
	{"lambda_logs", "collect_logs_cloudwatch", isLambdaForwarder, "sumologic_cloudwatch_logs_source"},
}

func isBoolTrue(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "false" && val != "False"
	}
	return true
}

func isKFLogs(v interface{}) bool       { s, _ := v.(string); return s == "Kinesis Firehose Log Source" }
func isKFMetrics(v interface{}) bool    { s, _ := v.(string); return s == "Kinesis Firehose Metrics Source" }
func isLambdaForwarder(v interface{}) bool { s, _ := v.(string); return s == "Lambda Log Forwarder" }

// validateSumoDataFlow validates only the sources created by THIS test have data.
func validateSumoDataFlow(t *testing.T, vars map[string]interface{}, workingDir string) {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	collectorID := terraform.Output(t, opts, "sumologic_collector")
	if collectorID == "" {
		t.Log("[e2e] No collector output — skipping Sumo data flow validation")
		return
	}

	search := &testresources.SumoSearchClient{Cfg: sumoSearchCfg()}
	collectorName, err := search.GetCollectorName(collectorID)
	if err != nil {
		t.Errorf("[e2e] Failed to get collector name: %v", err)
		return
	}

	type toValidate struct {
		label    string
		sourceID string
	}
	var sources []toValidate

	for _, sk := range sourceOutputKeys {
		varVal, exists := vars[sk.varKey]
		if !exists || !sk.matchFn(varVal) {
			continue
		}
		sourceID := terraform.Output(t, opts, sk.outputKey)
		if sourceID == "" {
			continue
		}
		sources = append(sources, toValidate{label: sk.label, sourceID: sourceID})
	}

	if len(sources) == 0 {
		t.Log("[e2e] No source outputs to validate")
		return
	}

	t.Logf("[e2e] Validating %d sources under collector %q", len(sources), collectorName)
	passed, failed := 0, 0
	for _, s := range sources {
		result := search.ValidateSourceHasData(t, collectorID, s.sourceID, collectorName, e2eRetryDelays)
		if result.Error != nil {
			t.Errorf("[e2e] %s: error validating source %s: %v", s.label, s.sourceID, result.Error)
			failed++
		} else if result.HasData {
			t.Logf("[e2e] PASS: %s source %q has data", s.label, result.Source.Name)
			passed++
		} else {
			t.Errorf("[e2e] FAIL: %s source %q has NO data — query: %s", s.label, result.Source.Name, result.Query)
			failed++
		}
	}
	t.Logf("[e2e] Sumo data flow: %d/%d sources passed", passed, len(sources))
}

func getBoolVar(vars map[string]interface{}, key string, defaultVal bool) bool {
	v, ok := vars[key]
	if !ok {
		return defaultVal
	}
	return isBoolTrue(v)
}

func getStrVar(vars map[string]interface{}, key, defaultVal string) string {
	v, ok := vars[key]
	if !ok {
		return defaultVal
	}
	s, _ := v.(string)
	return s
}

// validateS3BucketTags fetches S3 bucket tags and asserts all expected tags are present.
// Returns actual tag map for further assertions.
func validateS3BucketTags(t *testing.T, bucketName string, expectedTags map[string]string) map[string]string {
	t.Helper()
	out := runShell(fmt.Sprintf(
		`aws s3api get-bucket-tagging --bucket "%s" --region %s --query 'TagSet' --output json 2>/dev/null`,
		bucketName, region,
	))
	actual := parseTagsJSON(out)
	for k, v := range expectedTags {
		if got, ok := actual[k]; !ok {
			t.Errorf("[tags] S3 bucket %s missing tag %q", bucketName, k)
		} else if got != v {
			t.Errorf("[tags] S3 bucket %s tag %q = %q, want %q", bucketName, k, got, v)
		}
	}
	return actual
}

// validateIAMRoleTags fetches IAM role tags and asserts all expected tags are present.
func validateIAMRoleTags(t *testing.T, roleName string, expectedTags map[string]string) map[string]string {
	t.Helper()
	out := runShell(fmt.Sprintf(
		`aws iam list-role-tags --role-name "%s" --query 'Tags' --output json 2>/dev/null`,
		roleName,
	))
	actual := parseTagsJSON(out)
	for k, v := range expectedTags {
		if got, ok := actual[k]; !ok {
			t.Errorf("[tags] IAM role %s missing tag %q", roleName, k)
		} else if got != v {
			t.Errorf("[tags] IAM role %s tag %q = %q, want %q", roleName, k, got, v)
		}
	}
	return actual
}

// validateKinesisFirehoseTags fetches Firehose stream tags and asserts expected tags are present.
func validateKinesisFirehoseTags(t *testing.T, streamName string, expectedTags map[string]string) {
	t.Helper()
	out := runShell(fmt.Sprintf(
		`aws firehose list-tags-for-delivery-stream --delivery-stream-name "%s" --region %s --query 'Tags' --output json 2>/dev/null`,
		streamName, region,
	))
	actual := parseTagsJSON(out)
	for k, v := range expectedTags {
		if got, ok := actual[k]; !ok {
			t.Errorf("[tags] KF stream %s missing tag %q", streamName, k)
		} else if got != v {
			t.Errorf("[tags] KF stream %s tag %q = %q, want %q", streamName, k, got, v)
		}
	}
}

// validateCloudTrailTags fetches CloudTrail trail tags and asserts expected tags are present.
func validateCloudTrailTags(t *testing.T, trailName string, expectedTags map[string]string) {
	t.Helper()
	out := runShell(fmt.Sprintf(
		`aws cloudtrail list-tags --resource-id-list "%s" --region %s --query 'ResourceTagList[0].TagsList' --output json 2>/dev/null`,
		trailName, region,
	))
	actual := parseTagsJSON(out)
	for k, v := range expectedTags {
		if got, ok := actual[k]; !ok {
			t.Errorf("[tags] CloudTrail %s missing tag %q", trailName, k)
		} else if got != v {
			t.Errorf("[tags] CloudTrail %s tag %q = %q, want %q", trailName, k, got, v)
		}
	}
}

// parseTagsJSON parses AWS CLI tag list output: [{"Key":"k","Value":"v"}, ...] or [{"key":"k","value":"v"}, ...]
func parseTagsJSON(jsonStr string) map[string]string {
	result := make(map[string]string)
	if jsonStr == "" || jsonStr == "null" {
		return result
	}
	// Quick line-based parser to avoid importing encoding/json in helpers
	lines := strings.Split(jsonStr, "\n")
	var curKey, curVal string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if k, ok := extractJSONString(line, "Key"); ok {
			curKey = k
		} else if k, ok := extractJSONString(line, "key"); ok {
			curKey = k
		}
		if v, ok := extractJSONString(line, "Value"); ok {
			curVal = v
		} else if v, ok := extractJSONString(line, "value"); ok {
			curVal = v
		}
		if curKey != "" && curVal != "" {
			result[curKey] = curVal
			curKey, curVal = "", ""
		}
	}
	return result
}

// putBucketPolicy replaces the bucket policy with the provided JSON string.
func putBucketPolicy(t *testing.T, bucketName, policyJSON string) {
	t.Helper()
	out := runShell(fmt.Sprintf(
		`aws s3api put-bucket-policy --bucket "%s" --region %s --policy '%s' 2>&1`,
		bucketName, region, policyJSON,
	))
	if out != "" {
		t.Fatalf("[policy] put-bucket-policy on %s failed: %s", bucketName, out)
	}
	t.Logf("[policy] applied custom policy to bucket %s", bucketName)
}

// assertBucketPolicyContains fetches the S3 bucket policy and fails if expected is absent.
func assertBucketPolicyContains(t *testing.T, bucketName, expected string) {
	t.Helper()
	policy := runShell(fmt.Sprintf(
		`aws s3api get-bucket-policy --bucket "%s" --region %s --query Policy --output text 2>/dev/null`,
		bucketName, region,
	))
	if !strings.Contains(policy, expected) {
		t.Errorf("[policy] bucket %s: policy does not contain %q\nactual: %s", bucketName, expected, policy)
	} else {
		t.Logf("[policy] bucket %s: contains %q (OK)", bucketName, expected)
	}
}

// assertBucketHasSNSNotification fails if the bucket has no S3 ObjectCreated topic notification.
func assertBucketHasSNSNotification(t *testing.T, bucketName string) {
	t.Helper()
	out := runShell(fmt.Sprintf(
		`aws s3api get-bucket-notification-configuration --bucket "%s" --region %s --query 'TopicConfigurations | length(@)' --output text 2>/dev/null`,
		bucketName, region,
	))
	count := strings.TrimSpace(out)
	if count == "" || count == "0" || count == "None" {
		t.Errorf("[sns] bucket %s: no SNS TopicConfiguration found (S3→SNS notification not wired)", bucketName)
	} else {
		t.Logf("[sns] bucket %s: %s SNS notification(s) configured (OK)", bucketName, count)
	}
}

// assertIAMRoleExists fails if the given IAM role name cannot be found.
func assertIAMRoleExists(t *testing.T, roleName string) {
	t.Helper()
	if roleName == "" {
		t.Log("[iam] no role name provided — skipping IAM role existence check")
		return
	}
	out := runShell(fmt.Sprintf(
		`aws iam get-role --role-name "%s" --query 'Role.RoleName' --output text 2>/dev/null`,
		roleName,
	))
	if strings.TrimSpace(out) != roleName {
		t.Errorf("[iam] IAM role %q not found (got %q)", roleName, out)
	} else {
		t.Logf("[iam] IAM role %q exists (OK)", roleName)
	}
}

func extractJSONString(line, key string) (string, bool) {
	prefix := fmt.Sprintf(`"%s":`, key)
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return "", false
	}
	rest := strings.TrimSpace(line[idx+len(prefix):])
	rest = strings.Trim(rest, `",`)
	return rest, true
}

// assertBucketForceDestroy reads the terraform state for the common S3 bucket and verifies
// that force_destroy matches the expected value. Catches the OR-logic regression at state level.
func assertBucketForceDestroy(t *testing.T, workingDir string, expected bool) {
	t.Helper()
	opts := test_structure.LoadTerraformOptions(t, workingDir)
	out := terraform.RunTerraformCommand(t, opts, "state", "show",
		`module.collection-module.aws_s3_bucket.s3_bucket["s3_bucket"]`)
	// Parse each line looking for "force_destroy" — spacing varies by Terraform version
	actual := false
	found := false
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "force_destroy") {
			found = true
			actual = strings.Contains(trimmed, "true")
			break
		}
	}
	if !found {
		t.Fatalf("[state] force_destroy not found in bucket state output:\n%s", out)
	}
	if actual != expected {
		t.Errorf("[state] expected force_destroy=%v in bucket state, but got %v", expected, actual)
	} else {
		t.Logf("[state] force_destroy=%v confirmed in bucket state (OK)", expected)
	}
}
