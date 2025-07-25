package storage

import (
	"fmt"
	"path/filepath"
	"strings"

	apptypes "github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// MockS3Client is a mock implementation of StorageClient for testing
type MockS3Client struct {
	config        *apptypes.MigrationConfig
	buckets       map[string]bool
	objects       map[string]map[string]bool // bucket -> key -> exists
	uploadResults map[string]string          // localPath -> uploadedURL
	shouldFail    map[string]bool            // operation -> should fail
}

// NewMockS3Client creates a new mock S3 client for testing
func NewMockS3Client(migrationConfig *apptypes.MigrationConfig) *MockS3Client {
	return &MockS3Client{
		config:        migrationConfig,
		buckets:       make(map[string]bool),
		objects:       make(map[string]map[string]bool),
		uploadResults: make(map[string]string),
		shouldFail:    make(map[string]bool),
	}
}

// SetBucketExists simulates bucket existence for testing
func (m *MockS3Client) SetBucketExists(bucket string, exists bool) {
	m.buckets[bucket] = exists
}

// SetObjectExists simulates object existence for testing
func (m *MockS3Client) SetObjectExists(bucket, key string, exists bool) {
	if m.objects[bucket] == nil {
		m.objects[bucket] = make(map[string]bool)
	}
	m.objects[bucket][key] = exists
}

// SetUploadResult simulates upload results for testing
func (m *MockS3Client) SetUploadResult(localPath, uploadedURL string) {
	m.uploadResults[localPath] = uploadedURL
}

// SetShouldFail configures operations to fail for testing error conditions
func (m *MockS3Client) SetShouldFail(operation string, shouldFail bool) {
	m.shouldFail[operation] = shouldFail
}

// BucketExists checks if the specified bucket exists (mock implementation)
func (m *MockS3Client) BucketExists(bucket string) (bool, error) {
	if m.shouldFail["BucketExists"] {
		return false, fmt.Errorf("mock error: bucket existence check failed")
	}

	exists, found := m.buckets[bucket]
	if !found {
		return false, nil // Default to not existing
	}
	return exists, nil
}

// CreateBucket creates a new bucket (mock implementation)
func (m *MockS3Client) CreateBucket(bucket, region string) error {
	if m.shouldFail["CreateBucket"] {
		return fmt.Errorf("mock error: failed to create bucket %s", bucket)
	}

	// Check if bucket already exists
	if exists, _ := m.BucketExists(bucket); exists {
		return fmt.Errorf("bucket already exists: %s", bucket)
	}

	m.buckets[bucket] = true
	return nil
}

// UploadFile uploads a file to storage (mock implementation)
func (m *MockS3Client) UploadFile(localPath, bucket, key string) (string, error) {
	if m.shouldFail["UploadFile"] {
		return "", fmt.Errorf("mock error: failed to upload file %s", localPath)
	}

	// Check if file exists (in real implementation)
	if !strings.HasPrefix(localPath, "/test/") && !filepath.IsAbs(localPath) {
		// For non-test paths, simulate file not found
		return "", fmt.Errorf("file not found: %s", localPath)
	}

	// Check if bucket exists
	exists, err := m.BucketExists(bucket)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("bucket does not exist: %s", bucket)
	}

	// Mark object as existing
	m.SetObjectExists(bucket, key, true)

	// Return predefined URL or generate one
	if url, found := m.uploadResults[localPath]; found {
		return url, nil
	}

	// Generate mock URL
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, m.config.S3Region, key), nil
}

// GetPublicURL returns the public URL for an object (mock implementation)
func (m *MockS3Client) GetPublicURL(bucket, key string) string {
	if m.shouldFail["GetPublicURL"] {
		return ""
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, m.config.S3Region, key)
}

// ObjectExists checks if an object exists in storage (mock implementation)
func (m *MockS3Client) ObjectExists(bucket, key string) (bool, error) {
	if m.shouldFail["ObjectExists"] {
		return false, fmt.Errorf("mock error: object existence check failed")
	}

	bucketObjects, found := m.objects[bucket]
	if !found {
		return false, nil
	}

	exists, found := bucketObjects[key]
	if !found {
		return false, nil
	}

	return exists, nil
}

// ListObjects lists objects in a bucket with optional prefix (mock implementation)
func (m *MockS3Client) ListObjects(bucket, prefix string) ([]string, error) {
	if m.shouldFail["ListObjects"] {
		return nil, fmt.Errorf("mock error: failed to list objects")
	}

	bucketObjects, found := m.objects[bucket]
	if !found {
		return []string{}, nil
	}

	var objects []string
	for key, exists := range bucketObjects {
		if exists && (prefix == "" || strings.HasPrefix(key, prefix)) {
			objects = append(objects, key)
		}
	}

	return objects, nil
}

// DeleteObject deletes an object from storage (mock implementation)
func (m *MockS3Client) DeleteObject(bucket, key string) error {
	if m.shouldFail["DeleteObject"] {
		return fmt.Errorf("mock error: failed to delete object %s", key)
	}

	bucketObjects, found := m.objects[bucket]
	if !found {
		return fmt.Errorf("bucket not found: %s", bucket)
	}

	delete(bucketObjects, key)
	return nil
}

// Close closes the storage client connection (mock implementation)
func (m *MockS3Client) Close() error {
	// Nothing to close in mock
	return nil
}
