package storage

// StorageClient defines the interface for storage operations
type StorageClient interface {
	// BucketExists checks if the specified bucket exists
	BucketExists(bucket string) (bool, error)

	// CreateBucket creates a new bucket
	CreateBucket(bucket, region string) error

	// UploadFile uploads a file to storage
	UploadFile(localPath, bucket, key string) (string, error)

	// GetPublicURL returns the public URL for an object
	GetPublicURL(bucket, key string) string

	// ObjectExists checks if an object exists in storage
	ObjectExists(bucket, key string) (bool, error)

	// ListObjects lists objects in a bucket with optional prefix
	ListObjects(bucket, prefix string) ([]string, error)

	// DeleteObject deletes an object from storage
	DeleteObject(bucket, key string) error
}
