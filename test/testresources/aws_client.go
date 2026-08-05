package testresources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GetS3Client returns an S3 client for the given region.
func GetS3Client(t *testing.T, region string) *s3.Client {
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		t.Fatalf("[testresources] Failed to load AWS config: %v", err)
	}
	return s3.NewFromConfig(cfg)
}
