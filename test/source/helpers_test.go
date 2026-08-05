package source_test

import (
	"bufio"
	"fmt"
	"log"
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
			val = strings.Trim(val, `"`)
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

// deployTerraform runs init → apply, saves opts and returns resource counts.
func deployTerraform(t *testing.T, workingDir string, vars map[string]interface{}, varsFile string) *terraform.ResourceCount {
	t.Helper()
	varFiles := []string{}
	if varsFile != "" {
		varFiles = []string{varsFile}
	}
	return testresources.DeployTerraform(t, workingDir, vars, varFiles)
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

// destroyTerraform tears down and cleans state files.
func destroyTerraform(t *testing.T, workingDir string) {
	t.Helper()
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
