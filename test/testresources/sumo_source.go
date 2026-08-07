package testresources

import (
	"fmt"
	"testing"
)

// SumoSource creates/deletes a Sumo Logic S3 source on an existing collector.
// Used to pre-create a source whose URL can be passed as *_log_source_url variable.
type SumoSource struct {
	Cfg         Config
	CollectorID string
	Name        string
	Category    string
	BucketName  string
	id          string
}

func (s *SumoSource) Create(t *testing.T) string {
	payload := fmt.Sprintf(
		`{"source":{"sourceType":"Polling","contentType":"AwsS3Bucket","name":"%s","category":"%s","thirdPartyRef":{"resources":[{"serviceType":"AwsS3Bucket","path":{"type":"S3BucketPathExpression","bucketName":"%s","pathExpression":"*"},"authentication":{"type":"AWSRoleBasedAuthentication","roleARN":"arn:aws:iam::000000000000:role/placeholder"}}]}}}`,
		s.Name, s.Category, s.BucketName,
	)
	id := shellOutput(fmt.Sprintf(
		`curl -s -u %s -X POST -H "Content-Type: application/json" -d '%s' "%s/api/v1/collectors/%s/sources" | jq -r '.source.id // empty'`,
		s.Cfg.sumoCreds(), payload, s.Cfg.SumoBaseURL, s.CollectorID,
	))
	if id == "" || id == "null" {
		t.Fatalf("[testresources] Failed to create Sumo source %q on collector %s", s.Name, s.CollectorID)
	}
	t.Logf("[testresources] Created Sumo source name=%s id=%s on collector %s", s.Name, id, s.CollectorID)
	s.id = id
	return id
}

func (s *SumoSource) Delete(t *testing.T) {
	if s.id == "" {
		return
	}
	shellOutput(fmt.Sprintf(
		`curl -sf -u %s -X DELETE "%s/api/v1/collectors/%s/sources/%s"`,
		s.Cfg.sumoCreds(), s.Cfg.SumoBaseURL, s.CollectorID, s.id,
	))
	t.Logf("[testresources] Deleted Sumo source id=%s", s.id)
}

func (s *SumoSource) ID() string { return s.id }

// SourceURL returns the Sumo API URL used as *_log_source_url variable values.
func (s *SumoSource) SourceURL() string {
	return fmt.Sprintf("%s/api/v1/collectors/%s/sources/%s", s.Cfg.SumoBaseURL, s.CollectorID, s.id)
}
