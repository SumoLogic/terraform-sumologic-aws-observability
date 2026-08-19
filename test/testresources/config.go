package testresources

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

// Config holds credentials and endpoint information for creating test prerequisites.
// Values are sourced from environment variables, falling back to a tfvars file.
type Config struct {
	SumoBaseURL   string // e.g. "https://api.us2.sumologic.com"
	SumoAccessID  string
	SumoAccessKey string
	AWSRegion     string // defaults to "us-east-1"
}

func (c Config) region() string {
	if c.AWSRegion == "" {
		return "us-east-1"
	}
	return c.AWSRegion
}

func (c Config) sumoCreds() string {
	return c.SumoAccessID + ":" + c.SumoAccessKey
}

// LoadConfig builds a Config from environment variables with a fallback to a tfvars file.
// Env vars: SUMO_ACCESS_ID, SUMO_ACCESS_KEY, SUMO_API_ENDPOINT, AWS_DEFAULT_REGION.
func LoadConfig(tfvarsPath string) Config {
	props := readTFVars(tfvarsPath)

	accessID := envOrProp("SUMO_ACCESS_ID", "sumologic_access_id", props)
	accessKey := envOrProp("SUMO_ACCESS_KEY", "sumologic_access_key", props)
	apiEndpoint := envOrProp("SUMO_API_ENDPOINT", "sumo_api_endpoint", props)
	region := envOrProp("AWS_DEFAULT_REGION", "", props)
	if region == "" {
		region = "us-east-1"
	}

	baseURL := ""
	if apiEndpoint != "" {
		u, err := url.Parse(apiEndpoint)
		if err == nil {
			baseURL = u.Scheme + "://" + u.Host
		}
	}

	return Config{
		SumoBaseURL:   baseURL,
		SumoAccessID:  accessID,
		SumoAccessKey: accessKey,
		AWSRegion:     region,
	}
}

func envOrProp(envKey, propKey string, props map[string]string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if propKey != "" {
		return props[propKey]
	}
	return ""
}

// readTFVars reads a key = "value" file (terraform.tfvars / main.auto.tfvars style).
func readTFVars(path string) map[string]string {
	result := make(map[string]string)
	if path == "" {
		return result
	}
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[config] Warning: could not open %s: %v", path, err)
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

// RandHex returns 4 random hex bytes (8 chars) using openssl, for unique resource names.
func RandHex() string {
	out := shellOutput(`openssl rand -hex 4`)
	if out == "" {
		return fmt.Sprintf("%08x", os.Getpid())
	}
	return out
}
