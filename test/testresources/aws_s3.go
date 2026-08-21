package testresources

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var protectedBucketPrefixes = []string{"appdevzipfiles"}
var protectedBucketNames = []string{"sumologic-appdev-aws-sam-apps"}

func isProtectedBucket(name string) bool {
	for _, exact := range protectedBucketNames {
		if name == exact {
			return true
		}
	}
	for _, prefix := range protectedBucketPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// AWSS3Bucket creates/deletes an S3 bucket as a test prerequisite.
type AWSS3Bucket struct {
	Cfg  Config
	Name string // optional; auto-generated with "awso-test-" prefix if empty
	name string // resolved after Create
}

func (b *AWSS3Bucket) Create(t *testing.T) string {
	name := b.Name
	if name == "" {
		name = "awso-test-" + RandHex()
	}
	client := GetS3Client(t, b.Cfg.region())
	_, err := client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		t.Fatalf("[testresources] Failed to create S3 bucket %s: %v", name, err)
	}
	b.name = name
	t.Logf("[testresources] Created S3 bucket %s", name)

	b.disableBlockPublicAccess(t, client)
	b.attachBucketPolicy(t, client)
	return name
}

func (b *AWSS3Bucket) disableBlockPublicAccess(t *testing.T, client *s3.Client) {
	f := false
	_, err := client.PutPublicAccessBlock(context.TODO(), &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(b.name),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       &f,
			IgnorePublicAcls:      &f,
			BlockPublicPolicy:     &f,
			RestrictPublicBuckets: &f,
		},
	})
	if err != nil {
		t.Logf("[testresources] Warning: disable block-public-access on %s: %v", b.name, err)
	}
	time.Sleep(3 * time.Second)
}

func (b *AWSS3Bucket) attachBucketPolicy(t *testing.T, client *s3.Client) {
	policy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowAllRead",
      "Effect": "Allow",
      "Principal": "*",
      "Action": ["s3:GetObject","s3:GetObjectVersion","s3:ListBucketVersions","s3:ListBucket","s3:GetBucketAcl","s3:GetBucketLocation"],
      "Resource": ["arn:aws:s3:::%s/*","arn:aws:s3:::%s"]
    },
    {
      "Sid": "AllowServiceWrite",
      "Effect": "Allow",
      "Principal": "*",
      "Action": ["s3:PutObject"],
      "Resource": ["arn:aws:s3:::%s/*"]
    }
  ]
}`, b.name, b.name, b.name)

	_, err := client.PutBucketPolicy(context.TODO(), &s3.PutBucketPolicyInput{
		Bucket: aws.String(b.name),
		Policy: aws.String(policy),
	})
	if err != nil {
		t.Logf("[testresources] Warning: attach bucket policy to %s: %v", b.name, err)
	} else {
		time.Sleep(10 * time.Second)
	}
}

func (b *AWSS3Bucket) Delete(t *testing.T) {
	if b.name == "" {
		return
	}
	if isProtectedBucket(b.name) {
		t.Logf("[testresources] SKIPPING delete of protected bucket %s", b.name)
		return
	}
	client := GetS3Client(t, b.Cfg.region())
	b.emptyBucket(t, client)
	_, err := client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String(b.name)})
	if err != nil {
		t.Logf("[testresources] Warning: delete bucket %s: %v", b.name, err)
		return
	}
	t.Logf("[testresources] Deleted S3 bucket %s", b.name)
}

// ForceDelete empties and deletes the bucket even when force_destroy_bucket=false prevented
// Terraform from removing it. Used in cleanup for TestFeature_BucketRetention.
func (b *AWSS3Bucket) ForceDelete(t *testing.T) {
	name := b.name
	if name == "" {
		name = b.Name
	}
	if name == "" || isProtectedBucket(name) {
		return
	}
	client := GetS3Client(t, b.Cfg.region())
	b.name = name
	b.emptyBucket(t, client)
	_, err := client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: aws.String(name)})
	if err != nil {
		t.Logf("[testresources] ForceDelete: bucket %s: %v", name, err)
		return
	}
	t.Logf("[testresources] ForceDeleted S3 bucket %s", name)
}

// AssertExists fails the test if the bucket does not exist (used by bucket-retention test).
func (b *AWSS3Bucket) AssertExists(t *testing.T) {
	name := b.name
	if name == "" {
		name = b.Name
	}
	client := GetS3Client(t, b.Cfg.region())
	_, err := client.HeadBucket(context.TODO(), &s3.HeadBucketInput{Bucket: aws.String(name)})
	if err != nil {
		t.Errorf("[testresources] AssertExists: bucket %s not found after destroy (retention check failed): %v", name, err)
	} else {
		t.Logf("[testresources] AssertExists: bucket %s still exists after destroy (retention OK)", name)
	}
}

func (b *AWSS3Bucket) ID() string { return b.name }

// PutObject uploads a small object to the bucket. Useful for making a bucket
// non-empty so that force_destroy=false prevents deletion on terraform destroy.
func (b *AWSS3Bucket) PutObject(t *testing.T, key string) {
	name := b.name
	if name == "" {
		name = b.Name
	}
	client := GetS3Client(t, b.Cfg.region())
	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(name),
		Key:    aws.String(key),
		Body:   strings.NewReader("placeholder"),
	})
	if err != nil {
		t.Fatalf("[testresources] PutObject %s/%s failed: %v", name, key, err)
	}
	t.Logf("[testresources] PutObject: placed %s in bucket %s", key, name)
}

// AssertObjectExists fails if the given key does not exist in the bucket.
func (b *AWSS3Bucket) AssertObjectExists(t *testing.T, key string) {
	t.Helper()
	name := b.name
	if name == "" {
		name = b.Name
	}
	client := GetS3Client(t, b.Cfg.region())
	_, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(name),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Errorf("[testresources] AssertObjectExists: %s/%s not found: %v", name, key, err)
	} else {
		t.Logf("[testresources] AssertObjectExists: %s/%s exists (OK)", name, key)
	}
}

// AssertNotExists fails if the bucket still exists (used to verify force_destroy=true cleanup).
func (b *AWSS3Bucket) AssertNotExists(t *testing.T) {
	t.Helper()
	name := b.name
	if name == "" {
		name = b.Name
	}
	client := GetS3Client(t, b.Cfg.region())
	_, err := client.HeadBucket(context.TODO(), &s3.HeadBucketInput{Bucket: aws.String(name)})
	if err == nil {
		t.Errorf("[testresources] AssertNotExists: bucket %s still exists (expected deletion)", name)
	} else {
		t.Logf("[testresources] AssertNotExists: bucket %s confirmed deleted (OK)", name)
	}
}

func (b *AWSS3Bucket) emptyBucket(t *testing.T, client *s3.Client) {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{Bucket: aws.String(b.name)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			t.Logf("[testresources] Warning: list objects in %s: %v", b.name, err)
			return
		}
		if len(page.Contents) == 0 {
			continue
		}
		objects := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = types.ObjectIdentifier{Key: obj.Key}
		}
		client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: aws.String(b.name),
			Delete: &types.Delete{Objects: objects},
		})
	}
}
