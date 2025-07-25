package mocks

import "github.com/SynapsesTechnologies/mv2s3/internal/storage"

// MockStorageClient implements StorageClient for testing
type MockStorageClient struct {
	BucketExistsFunc func(bucket string) (bool, error)
	CreateBucketFunc func(bucket, region string) error
	UploadFileFunc   func(localPath, bucket, key string) (string, error)
	GetPublicURLFunc func(bucket, key string) string
	ObjectExistsFunc func(bucket, key string) (bool, error)
	ListObjectsFunc  func(bucket, prefix string) ([]string, error)
	DeleteObjectFunc func(bucket, key string) error
}

// BucketExists mock implementation
func (m *MockStorageClient) BucketExists(bucket string) (bool, error) {
	if m.BucketExistsFunc != nil {
		return m.BucketExistsFunc(bucket)
	}
	return true, nil
}

// CreateBucket mock implementation
func (m *MockStorageClient) CreateBucket(bucket, region string) error {
	if m.CreateBucketFunc != nil {
		return m.CreateBucketFunc(bucket, region)
	}
	return nil
}

// UploadFile mock implementation
func (m *MockStorageClient) UploadFile(localPath, bucket, key string) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(localPath, bucket, key)
	}
	return "https://mock-bucket.s3.amazonaws.com/" + key, nil
}

// GetPublicURL mock implementation
func (m *MockStorageClient) GetPublicURL(bucket, key string) string {
	if m.GetPublicURLFunc != nil {
		return m.GetPublicURLFunc(bucket, key)
	}
	return "https://" + bucket + ".s3.amazonaws.com/" + key
}

// ObjectExists mock implementation
func (m *MockStorageClient) ObjectExists(bucket, key string) (bool, error) {
	if m.ObjectExistsFunc != nil {
		return m.ObjectExistsFunc(bucket, key)
	}
	return true, nil
}

// ListObjects mock implementation
func (m *MockStorageClient) ListObjects(bucket, prefix string) ([]string, error) {
	if m.ListObjectsFunc != nil {
		return m.ListObjectsFunc(bucket, prefix)
	}
	// Return some mock objects for testing
	return []string{
		prefix + "image1.jpg",
		prefix + "image2.png",
		prefix + "subfolder/image3.gif",
	}, nil
}

// DeleteObject mock implementation
func (m *MockStorageClient) DeleteObject(bucket, key string) error {
	if m.DeleteObjectFunc != nil {
		return m.DeleteObjectFunc(bucket, key)
	}
	return nil
}

// Verify MockStorageClient implements StorageClient interface
var _ storage.StorageClient = (*MockStorageClient)(nil)
