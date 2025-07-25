package storage

import (
	"testing"

	apptypes "github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// TestHelper provides utilities for testing with storage clients
type TestHelper struct {
	MockClient *MockS3Client
	Config     *apptypes.MigrationConfig
}

// NewTestHelper creates a new test helper with a mock S3 client
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	config := &apptypes.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
		S3Prefix: "test/",
	}

	mockClient := NewMockS3Client(config)

	// Set up default test state
	mockClient.SetBucketExists("test-bucket", true)

	return &TestHelper{
		MockClient: mockClient,
		Config:     config,
	}
}

// CreateTestBucket sets up a test bucket
func (th *TestHelper) CreateTestBucket(bucket string) {
	th.MockClient.SetBucketExists(bucket, true)
}

// CreateTestObject sets up a test object
func (th *TestHelper) CreateTestObject(bucket, key string) {
	th.MockClient.SetObjectExists(bucket, key, true)
}

// SetupUploadSuccess configures a successful upload scenario
func (th *TestHelper) SetupUploadSuccess(localPath, expectedURL string) {
	th.MockClient.SetUploadResult(localPath, expectedURL)
}

// SetupFailure configures an operation to fail
func (th *TestHelper) SetupFailure(operation string) {
	th.MockClient.SetShouldFail(operation, true)
}

// GetStorageClient returns the mock client as a StorageClient interface
func (th *TestHelper) GetStorageClient() StorageClient {
	return th.MockClient
}

// AssertObjectExists verifies that an object exists
func (th *TestHelper) AssertObjectExists(t *testing.T, bucket, key string) {
	t.Helper()

	exists, err := th.MockClient.ObjectExists(bucket, key)
	if err != nil {
		t.Errorf("Error checking object existence: %v", err)
	}
	if !exists {
		t.Errorf("Expected object %s/%s to exist", bucket, key)
	}
}

// AssertObjectNotExists verifies that an object does not exist
func (th *TestHelper) AssertObjectNotExists(t *testing.T, bucket, key string) {
	t.Helper()

	exists, err := th.MockClient.ObjectExists(bucket, key)
	if err != nil {
		t.Errorf("Error checking object existence: %v", err)
	}
	if exists {
		t.Errorf("Expected object %s/%s to not exist", bucket, key)
	}
}

// AssertBucketExists verifies that a bucket exists
func (th *TestHelper) AssertBucketExists(t *testing.T, bucket string) {
	t.Helper()

	exists, err := th.MockClient.BucketExists(bucket)
	if err != nil {
		t.Errorf("Error checking bucket existence: %v", err)
	}
	if !exists {
		t.Errorf("Expected bucket %s to exist", bucket)
	}
}
