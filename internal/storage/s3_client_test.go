package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestNewS3Client(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	// Test that we can create a real S3 client (this should not fail)
	client, err := NewS3Client(config)
	if err != nil {
		t.Errorf("Expected no error creating S3 client, got: %v", err)
	}

	if client == nil {
		t.Error("Expected NewS3Client to return a non-nil client")
		return
	}

	if client.config != config {
		t.Error("Expected client config to match provided config")
	}
}

func TestNewMockS3Client(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)
	if client == nil {
		t.Error("Expected NewMockS3Client to return a non-nil client")
		return
	}

	if client.config != config {
		t.Error("Expected client config to match provided config")
	}
}

func TestS3Client_BucketExists(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Test bucket that doesn't exist
	exists, err := client.BucketExists("test-bucket")
	if err != nil {
		t.Errorf("Expected no error from BucketExists, got: %v", err)
	}
	if exists != false {
		t.Error("Expected BucketExists to return false for non-existent bucket")
	}

	// Test bucket that exists
	client.SetBucketExists("test-bucket", true)
	exists, err = client.BucketExists("test-bucket")
	if err != nil {
		t.Errorf("Expected no error from BucketExists, got: %v", err)
	}
	if exists != true {
		t.Error("Expected BucketExists to return true for existing bucket")
	}

	// Test error condition
	client.SetShouldFail("BucketExists", true)
	_, err = client.BucketExists("test-bucket")
	if err == nil {
		t.Error("Expected error from BucketExists when configured to fail")
	}
}

func TestS3Client_CreateBucket(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Test creating a new bucket
	err := client.CreateBucket("test-bucket", "us-east-1")
	if err != nil {
		t.Errorf("Expected no error from CreateBucket, got: %v", err)
	}

	// Verify bucket was created
	exists, err := client.BucketExists("test-bucket")
	if err != nil {
		t.Errorf("Expected no error checking bucket existence, got: %v", err)
	}
	if !exists {
		t.Error("Expected bucket to exist after creation")
	}

	// Test creating bucket that already exists
	err = client.CreateBucket("test-bucket", "us-east-1")
	if err == nil {
		t.Error("Expected error when creating bucket that already exists")
	}

	// Test error condition
	client.SetShouldFail("CreateBucket", true)
	err = client.CreateBucket("another-bucket", "us-east-1")
	if err == nil {
		t.Error("Expected error from CreateBucket when configured to fail")
	}
}

func TestS3Client_UploadFile(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Setup bucket
	client.SetBucketExists("test-bucket", true)

	// Create a temporary test file
	tempDir, err := os.MkdirTemp("", "s3-upload-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.jpg")
	testContent := []byte("fake image content")
	err = os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test successful upload
	url, err := client.UploadFile(testFile, "test-bucket", "images/test.jpg")
	if err != nil {
		t.Errorf("Expected no error from UploadFile, got: %v", err)
	}

	expectedURL := "https://test-bucket.s3.us-east-1.amazonaws.com/images/test.jpg"
	if url != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, url)
	}

	// Verify object was marked as existing
	exists, err := client.ObjectExists("test-bucket", "images/test.jpg")
	if err != nil {
		t.Errorf("Expected no error checking object existence, got: %v", err)
	}
	if !exists {
		t.Error("Expected object to exist after upload")
	}

	// Test upload to non-existent bucket
	_, err = client.UploadFile(testFile, "non-existent-bucket", "images/test.jpg")
	if err == nil {
		t.Error("Expected error when uploading to non-existent bucket")
	}

	// Test error condition
	client.SetShouldFail("UploadFile", true)
	_, err = client.UploadFile(testFile, "test-bucket", "images/test2.jpg")
	if err == nil {
		t.Error("Expected error from UploadFile when configured to fail")
	}
}

func TestS3Client_GetPublicURL(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Test getting public URL
	url := client.GetPublicURL("test-bucket", "images/file.jpg")
	expectedURL := "https://test-bucket.s3.us-east-1.amazonaws.com/images/file.jpg"
	if url != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, url)
	}

	// Test error condition
	client.SetShouldFail("GetPublicURL", true)
	url = client.GetPublicURL("test-bucket", "images/file.jpg")
	if url != "" {
		t.Errorf("Expected empty URL when configured to fail, got: %s", url)
	}
}

func TestMockS3Client_ObjectExists(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Test object that doesn't exist
	exists, err := client.ObjectExists("test-bucket", "images/file.jpg")
	if err != nil {
		t.Errorf("Expected no error from ObjectExists, got: %v", err)
	}
	if exists {
		t.Error("Expected ObjectExists to return false for non-existent object")
	}

	// Test object that exists
	client.SetObjectExists("test-bucket", "images/file.jpg", true)
	exists, err = client.ObjectExists("test-bucket", "images/file.jpg")
	if err != nil {
		t.Errorf("Expected no error from ObjectExists, got: %v", err)
	}
	if !exists {
		t.Error("Expected ObjectExists to return true for existing object")
	}

	// Test error condition
	client.SetShouldFail("ObjectExists", true)
	_, err = client.ObjectExists("test-bucket", "images/file.jpg")
	if err == nil {
		t.Error("Expected error from ObjectExists when configured to fail")
	}
}

func TestMockS3Client_ListObjects(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Setup some objects
	client.SetObjectExists("test-bucket", "images/file1.jpg", true)
	client.SetObjectExists("test-bucket", "images/file2.jpg", true)
	client.SetObjectExists("test-bucket", "docs/file1.pdf", true)

	// Test listing all objects
	objects, err := client.ListObjects("test-bucket", "")
	if err != nil {
		t.Errorf("Expected no error from ListObjects, got: %v", err)
	}
	if len(objects) != 3 {
		t.Errorf("Expected 3 objects, got %d", len(objects))
	}

	// Test listing with prefix
	objects, err = client.ListObjects("test-bucket", "images/")
	if err != nil {
		t.Errorf("Expected no error from ListObjects, got: %v", err)
	}
	if len(objects) != 2 {
		t.Errorf("Expected 2 objects with prefix 'images/', got %d", len(objects))
	}

	// Test error condition
	client.SetShouldFail("ListObjects", true)
	_, err = client.ListObjects("test-bucket", "")
	if err == nil {
		t.Error("Expected error from ListObjects when configured to fail")
	}
}

func TestMockS3Client_DeleteObject(t *testing.T) {
	config := &types.MigrationConfig{
		S3Bucket: "test-bucket",
		S3Region: "us-east-1",
	}

	client := NewMockS3Client(config)

	// Setup object
	client.SetObjectExists("test-bucket", "images/file.jpg", true)

	// Verify object exists
	exists, err := client.ObjectExists("test-bucket", "images/file.jpg")
	if err != nil {
		t.Errorf("Expected no error checking object existence, got: %v", err)
	}
	if !exists {
		t.Error("Expected object to exist before deletion")
	}

	// Delete object
	err = client.DeleteObject("test-bucket", "images/file.jpg")
	if err != nil {
		t.Errorf("Expected no error from DeleteObject, got: %v", err)
	}

	// Verify object no longer exists
	exists, err = client.ObjectExists("test-bucket", "images/file.jpg")
	if err != nil {
		t.Errorf("Expected no error checking object existence, got: %v", err)
	}
	if exists {
		t.Error("Expected object to not exist after deletion")
	}

	// Test error condition
	client.SetShouldFail("DeleteObject", true)
	err = client.DeleteObject("test-bucket", "images/file2.jpg")
	if err == nil {
		t.Error("Expected error from DeleteObject when configured to fail")
	}
}
