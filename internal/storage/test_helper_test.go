package storage

import (
	"testing"
)

func TestTestHelper(t *testing.T) {
	helper := NewTestHelper(t)

	// Test that default setup works
	helper.AssertBucketExists(t, "test-bucket")

	// Test creating objects
	helper.CreateTestObject("test-bucket", "test/file.jpg")
	helper.AssertObjectExists(t, "test-bucket", "test/file.jpg")

	// Test that non-existent objects are properly not found
	helper.AssertObjectNotExists(t, "test-bucket", "non-existent.jpg")

	// Test getting storage client interface
	client := helper.GetStorageClient()
	if client == nil {
		t.Error("Expected GetStorageClient to return non-nil client")
	}

	// Test client interface works
	url := client.GetPublicURL("test-bucket", "test/file.jpg")
	expectedURL := "https://test-bucket.s3.us-east-1.amazonaws.com/test/file.jpg"
	if url != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, url)
	}
}

func TestTestHelper_FailureScenarios(t *testing.T) {
	helper := NewTestHelper(t)

	// Test failure setup
	helper.SetupFailure("UploadFile")

	client := helper.GetStorageClient()
	_, err := client.UploadFile("/test/file.jpg", "test-bucket", "test/file.jpg")
	if err == nil {
		t.Error("Expected error from UploadFile when configured to fail")
	}
}
