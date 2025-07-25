package storage

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	apptypes "github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// S3Client implements StorageClient for AWS S3
type S3Client struct {
	s3Client *s3.Client
	uploader *manager.Uploader
	config   *apptypes.MigrationConfig
}

// NewS3Client creates a new S3 client
func NewS3Client(migrationConfig *apptypes.MigrationConfig) (*S3Client, error) {
	ctx := context.TODO()

	// Configure AWS SDK
	var opts []func(*config.LoadOptions) error

	// Set region
	if migrationConfig.S3Region != "" {
		opts = append(opts, config.WithRegion(migrationConfig.S3Region))
	}

	// Set profile if specified
	if migrationConfig.S3Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(migrationConfig.S3Profile))
	}

	// Set custom endpoint for S3-compatible services
	if migrationConfig.S3Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					URL:           migrationConfig.S3Endpoint,
					SigningRegion: migrationConfig.S3Region,
				}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})
		opts = append(opts, config.WithEndpointResolverWithOptions(customResolver))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Override credentials if provided
	if migrationConfig.S3AccessKey != "" && migrationConfig.S3SecretKey != "" {
		cfg.Credentials = aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     migrationConfig.S3AccessKey,
				SecretAccessKey: migrationConfig.S3SecretKey,
			}, nil
		})
	}

	s3Client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(s3Client)

	return &S3Client{
		s3Client: s3Client,
		uploader: uploader,
		config:   migrationConfig,
	}, nil
}

// BucketExists checks if the specified bucket exists
func (s3c *S3Client) BucketExists(bucket string) (bool, error) {
	ctx := context.TODO()

	_, err := s3c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		// Check if it's a "not found" or "no such bucket" error
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NotFound", "NoSuchBucket":
				return false, nil
			}
		}
		// Other errors (like permission denied) are actual errors
		return false, fmt.Errorf("error checking bucket existence: %w", err)
	}

	return true, nil
}

// CreateBucket creates a new bucket
func (s3c *S3Client) CreateBucket(bucket, region string) error {
	ctx := context.TODO()

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// For regions other than us-east-1, we need to specify the location constraint
	if region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err := s3c.s3Client.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
	}

	return nil
}

// UploadFile uploads a file to S3
func (s3c *S3Client) UploadFile(localPath, bucket, key string) (string, error) {
	ctx := context.TODO()

	// Open the local file
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	// Get file info for metadata
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Determine content type
	contentType := mime.TypeByExtension(filepath.Ext(localPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload the file
	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"original-path": localPath,
			"file-size":     fmt.Sprintf("%d", fileInfo.Size()),
			"uploaded-by":   "mv2s3",
		},
	}

	result, err := s3c.uploader.Upload(ctx, uploadInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return result.Location, nil
}

// GetPublicURL returns the public URL for an S3 object
func (s3c *S3Client) GetPublicURL(bucket, key string) string {
	// Handle custom endpoints
	if s3c.config.S3Endpoint != "" {
		baseURL := strings.TrimSuffix(s3c.config.S3Endpoint, "/")
		return fmt.Sprintf("%s/%s/%s", baseURL, bucket, key)
	}

	// Standard S3 URL format
	region := s3c.config.S3Region
	if region == "" {
		region = "us-east-1"
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
}

// ObjectExists checks if an object exists in S3
func (s3c *S3Client) ObjectExists(bucket, key string) (bool, error) {
	ctx := context.TODO()

	_, err := s3c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a "not found" or "no such key" error
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NotFound", "NoSuchKey":
				return false, nil
			}
		}
		return false, fmt.Errorf("error checking object existence: %w", err)
	}

	return true, nil
}

// ListObjects lists objects in a bucket with optional prefix
func (s3c *S3Client) ListObjects(bucket, prefix string) ([]string, error) {
	ctx := context.TODO()

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	var objects []string
	paginator := s3.NewListObjectsV2Paginator(s3c.s3Client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key != nil {
				objects = append(objects, *obj.Key)
			}
		}
	}

	return objects, nil
}

// GenerateS3Key generates an S3 key from an image path and prefix
func (s3c *S3Client) GenerateS3Key(imagePath, prefix string) string {
	// Clean the image path
	cleanPath := strings.TrimPrefix(imagePath, "./")
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	// Handle ../ paths by removing them (flatten to current directory)
	cleanPath = strings.TrimPrefix(cleanPath, "../")

	// Combine with prefix if provided
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		return prefix + "/" + cleanPath
	}

	return cleanPath
}

// ValidateConfiguration validates the S3 configuration
func (s3c *S3Client) ValidateConfiguration() error {
	// Check if we can access AWS credentials
	ctx := context.TODO()

	// Try to get caller identity to verify credentials
	_, err := s3c.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String("test-bucket-that-should-not-exist-12345"),
	})

	// We expect this to fail, but it should fail with a proper AWS error
	// If we get a credentials error, that means our setup is wrong
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NoSuchBucket", "AccessDenied":
				// These are expected - means credentials are working
				return nil
			case "InvalidAccessKeyId", "SignatureDoesNotMatch", "TokenRefreshRequired":
				return fmt.Errorf("AWS credentials error: %s", apiErr.ErrorMessage())
			}
		}
		return fmt.Errorf("unable to validate AWS configuration: %w", err)
	}

	return nil
}

// DeleteObject deletes an object from S3
func (s3c *S3Client) DeleteObject(bucket, key string) error {
	ctx := context.TODO()

	_, err := s3c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}

	return nil
}
