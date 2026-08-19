package testresources

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// GenerateALBTraffic sends HTTP GET requests to an ALB DNS endpoint to produce access logs.
// Matches CF test's background traffic generator pattern.
func GenerateALBTraffic(t *testing.T, albDNS string, requestCount int) {
	if albDNS == "" {
		return
	}
	url := fmt.Sprintf("http://%s/", albDNS)
	client := &http.Client{Timeout: 5 * time.Second}
	sent := 0
	for i := 0; i < requestCount; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			sent++
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Logf("[e2e-traffic] ALB %s: sent %d/%d requests", albDNS, sent, requestCount)
}

// GenerateCLBTraffic sends HTTP GET requests to a Classic LB DNS endpoint.
func GenerateCLBTraffic(t *testing.T, clbDNS string, requestCount int) {
	if clbDNS == "" {
		return
	}
	url := fmt.Sprintf("http://%s/", clbDNS)
	client := &http.Client{Timeout: 5 * time.Second}
	sent := 0
	for i := 0; i < requestCount; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			sent++
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Logf("[e2e-traffic] CLB %s: sent %d/%d requests", clbDNS, sent, requestCount)
}

// GenerateCloudTrailTraffic makes benign AWS API calls that produce CloudTrail events.
func GenerateCloudTrailTraffic(t *testing.T, region string) {
	if region == "" {
		region = "us-east-1"
	}
	shellOutput(fmt.Sprintf(`aws sts get-caller-identity --region %s`, region))
	shellOutput(fmt.Sprintf(`aws iam list-account-aliases --region %s`, region))
	shellOutput(fmt.Sprintf(`aws s3api list-buckets --region %s`, region))
	t.Logf("[e2e-traffic] CloudTrail: emitted 3 API call events in %s", region)
}

// GenerateCWLogsTraffic puts log events into a test log group.
func GenerateCWLogsTraffic(t *testing.T, region, logGroupName string, eventCount int) {
	if region == "" {
		region = "us-east-1"
	}
	if logGroupName == "" {
		logGroupName = "/awso/test/e2e"
	}

	shellOutput(fmt.Sprintf(
		`aws logs create-log-group --log-group-name "%s" --region %s 2>/dev/null || true`,
		logGroupName, region,
	))
	shellOutput(fmt.Sprintf(
		`aws logs create-log-stream --log-group-name "%s" --log-stream-name "e2e-stream" --region %s 2>/dev/null || true`,
		logGroupName, region,
	))

	events := ""
	for i := 0; i < eventCount; i++ {
		ts := time.Now().UnixMilli()
		events += fmt.Sprintf(`{"timestamp":%d,"message":"awso-e2e-test-event-%d"} `, ts, i)
		time.Sleep(10 * time.Millisecond)
	}

	shellOutput(fmt.Sprintf(
		`aws logs put-log-events --log-group-name "%s" --log-stream-name "e2e-stream" --log-events %s --region %s`,
		logGroupName, events, region,
	))
	t.Logf("[e2e-traffic] CW Logs %s: put %d events in %s", logGroupName, eventCount, region)
}
