package testresources

import (
	"fmt"
	"testing"
)

// SumoCollector creates/deletes a hosted Sumo Logic collector as a test prerequisite.
type SumoCollector struct {
	Cfg  Config
	Name string
	id   string
}

func (c *SumoCollector) Create(t *testing.T) string {
	// Remove any stale collector with the same name from a previous failed run.
	existingID := shellOutput(fmt.Sprintf(
		`curl -s -u %s "%s/api/v1/collectors?filter=%s" | jq -r '.collectors[] | select(.name=="%s") | .id' | head -1`,
		c.Cfg.sumoCreds(), c.Cfg.SumoBaseURL, c.Name, c.Name,
	))
	if existingID != "" && existingID != "null" {
		t.Logf("[testresources] Removing stale collector name=%s id=%s", c.Name, existingID)
		shellOutput(fmt.Sprintf(
			`curl -s -u %s -X DELETE "%s/api/v1/collectors/%s"`,
			c.Cfg.sumoCreds(), c.Cfg.SumoBaseURL, existingID,
		))
	}

	id := shellOutput(fmt.Sprintf(
		`curl -s -u %s -X POST -H "Content-Type: application/json" `+
			`-d '{"collector":{"collectorType":"Hosted","name":"%s"}}' `+
			`"%s/api/v1/collectors" | jq -r '.collector.id // empty'`,
		c.Cfg.sumoCreds(), c.Name, c.Cfg.SumoBaseURL,
	))
	if id == "" || id == "null" {
		t.Fatalf("[testresources] Failed to create Sumo collector %q", c.Name)
	}
	t.Logf("[testresources] Created Sumo collector name=%s id=%s", c.Name, id)
	c.id = id
	return id
}

func (c *SumoCollector) Delete(t *testing.T) {
	if c.id == "" {
		return
	}
	shellOutput(fmt.Sprintf(
		`curl -sf -u %s -X DELETE "%s/api/v1/collectors/%s"`,
		c.Cfg.sumoCreds(), c.Cfg.SumoBaseURL, c.id,
	))
	t.Logf("[testresources] Deleted Sumo collector id=%s", c.id)
}

func (c *SumoCollector) ID() string { return c.id }
