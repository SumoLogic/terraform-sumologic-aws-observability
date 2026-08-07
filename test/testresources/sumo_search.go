package testresources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// SumoSearchClient wraps the Sumo Logic Search Job and Metrics APIs.
type SumoSearchClient struct {
	Cfg Config
}

type searchJobStatus struct {
	State        string `json:"state"`
	MessageCount int    `json:"messageCount"`
}

// SearchHasDataWithBackoff runs a search job and retries with the given delay schedule.
// Returns true if messageCount > 0 on any attempt.
func (s *SumoSearchClient) SearchHasDataWithBackoff(t *testing.T, query, timeRange string, retryDelays []time.Duration) bool {
	maxAttempts := len(retryDelays) + 1
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		t.Logf("[sumo-search] Attempt %d/%d: query=%q range=%s", attempt, maxAttempts, query, timeRange)
		count, err := s.runSearchJob(query, timeRange)
		if err != nil {
			t.Logf("[sumo-search] Attempt %d error: %v", attempt, err)
		} else if count > 0 {
			t.Logf("[sumo-search] Found %d messages on attempt %d", count, attempt)
			return true
		} else {
			t.Logf("[sumo-search] 0 messages on attempt %d", attempt)
		}
		if attempt < maxAttempts {
			t.Logf("[sumo-search] Waiting %v before retry...", retryDelays[attempt-1])
			time.Sleep(retryDelays[attempt-1])
		}
	}
	return false
}

// MetricsHasDataWithBackoff checks the Metrics API for data, retrying with backoff.
func (s *SumoSearchClient) MetricsHasDataWithBackoff(t *testing.T, query string, retryDelays []time.Duration) bool {
	maxAttempts := len(retryDelays) + 1
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		t.Logf("[metrics-api] Attempt %d/%d: query=%q", attempt, maxAttempts, query)
		if s.metricsHasData(t, query) {
			t.Logf("[metrics-api] Found metrics data on attempt %d", attempt)
			return true
		}
		if attempt < maxAttempts {
			t.Logf("[metrics-api] Waiting %v before retry...", retryDelays[attempt-1])
			time.Sleep(retryDelays[attempt-1])
		}
	}
	return false
}

// GetCollectorName fetches the collector name from Sumo API given a collector ID.
func (s *SumoSearchClient) GetCollectorName(collectorID string) (string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/collectors/%s", s.Cfg.SumoBaseURL, collectorID), nil)
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(s.Cfg.SumoAccessID, s.Cfg.SumoAccessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GET collector %s: HTTP %d", collectorID, resp.StatusCode)
	}
	var result struct {
		Collector struct{ Name string `json:"name"` } `json:"collector"`
	}
	json.Unmarshal(body, &result)
	return result.Collector.Name, nil
}

// SourceInfo represents a Sumo Logic source from the list sources API.
type SourceInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	SourceType  string `json:"sourceType"`
	ContentType string `json:"contentType"`
	Category    string `json:"category"`
}

// SourceValidationResult holds the validation outcome for a single source.
type SourceValidationResult struct {
	Source  SourceInfo
	Query   string
	HasData bool
	Error   error
}

// ValidateSourceHasData validates that a specific source (by ID) has data in Sumo Logic.
func (s *SumoSearchClient) ValidateSourceHasData(
	t *testing.T,
	collectorID, sourceID, collectorName string,
	retryDelays []time.Duration,
) SourceValidationResult {
	src, err := s.getSource(collectorID, sourceID)
	if err != nil {
		t.Logf("[source-validate] Failed to get source %s: %v", sourceID, err)
		return SourceValidationResult{Error: err}
	}

	query := fmt.Sprintf(`_source="%s" _collector="%s"`, src.Name, collectorName)
	var hasData bool

	if isMetricsSource(*src) {
		t.Logf("[source-validate] METRICS source %q query: %s", src.Name, query)
		hasData = s.MetricsHasDataWithBackoff(t, query, retryDelays)
	} else {
		t.Logf("[source-validate] LOG source %q query: %s", src.Name, query)
		hasData = s.SearchHasDataWithBackoff(t, query, "-60m", retryDelays)
	}

	return SourceValidationResult{Source: *src, Query: query, HasData: hasData}
}

func (s *SumoSearchClient) runSearchJob(query, timeRange string) (int, error) {
	now := time.Now().UTC()
	var fromTime time.Time
	if len(timeRange) > 1 && timeRange[0] == '-' {
		dur, err := time.ParseDuration(timeRange[1:])
		if err != nil {
			return 0, fmt.Errorf("parse time range %q: %w", timeRange, err)
		}
		fromTime = now.Add(-dur)
	} else {
		fromTime = now.Add(-60 * time.Minute)
	}

	body := fmt.Sprintf(`{"query":"%s","from":"%s","to":"%s","timeZone":"UTC"}`,
		escapeJSON(query),
		fromTime.Format("2006-01-02T15:04:05"),
		now.Format("2006-01-02T15:04:05"),
	)

	req, _ := http.NewRequest("POST", s.Cfg.SumoBaseURL+"/api/v1/search/jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(s.Cfg.SumoAccessID, s.Cfg.SumoAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 202 {
		return 0, fmt.Errorf("create job: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var created struct{ ID string `json:"id"` }
	if err := json.Unmarshal(respBody, &created); err != nil {
		return 0, err
	}
	jobID := created.ID

	defer s.deleteJob(jobID)

	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		req2, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/search/jobs/%s", s.Cfg.SumoBaseURL, jobID), nil)
		req2.Header.Set("Accept", "application/json")
		req2.SetBasicAuth(s.Cfg.SumoAccessID, s.Cfg.SumoAccessKey)
		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		body2, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()

		var status searchJobStatus
		json.Unmarshal(body2, &status)
		switch status.State {
		case "DONE GATHERING RESULTS":
			return status.MessageCount, nil
		case "CANCELLED", "FORCE PAUSED":
			return 0, fmt.Errorf("job %s: state=%s", jobID, status.State)
		}
		time.Sleep(5 * time.Second)
	}
	return 0, fmt.Errorf("job %s timed out", jobID)
}

func (s *SumoSearchClient) metricsHasData(t *testing.T, query string) bool {
	now := time.Now().UTC()
	from := now.Add(-60 * time.Minute)
	body := fmt.Sprintf(`{"query":[{"query":"%s","rowId":"A"}],"startTime":%d,"endTime":%d,"requestedDataPoints":600,"maxDataPoints":800}`,
		escapeJSON(query), from.UnixMilli(), now.UnixMilli())

	req, _ := http.NewRequest("POST", s.Cfg.SumoBaseURL+"/api/v1/metrics/results", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(s.Cfg.SumoAccessID, s.Cfg.SumoAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("[metrics-api] Request failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Logf("[metrics-api] HTTP %d", resp.StatusCode)
		return false
	}

	var result struct {
		Response []struct {
			Results []json.RawMessage `json:"results"`
		} `json:"response"`
	}
	json.Unmarshal(respBody, &result)
	for _, r := range result.Response {
		if len(r.Results) > 0 {
			return true
		}
	}
	return false
}

func (s *SumoSearchClient) getSource(collectorID, sourceID string) (*SourceInfo, error) {
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/collectors/%s/sources/%s", s.Cfg.SumoBaseURL, collectorID, sourceID), nil)
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(s.Cfg.SumoAccessID, s.Cfg.SumoAccessKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET source %s: HTTP %d", sourceID, resp.StatusCode)
	}
	var result struct {
		Source SourceInfo `json:"source"`
	}
	json.Unmarshal(body, &result)
	return &result.Source, nil
}

func (s *SumoSearchClient) deleteJob(jobID string) {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/search/jobs/%s", s.Cfg.SumoBaseURL, jobID), nil)
	req.SetBasicAuth(s.Cfg.SumoAccessID, s.Cfg.SumoAccessKey)
	http.DefaultClient.Do(req)
}

func isMetricsSource(src SourceInfo) bool {
	ct := strings.ToLower(src.ContentType)
	st := strings.ToLower(src.SourceType)
	nm := strings.ToLower(src.Name)
	return strings.Contains(ct, "metric") || strings.Contains(st, "metric") || strings.Contains(nm, "metrics")
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}
