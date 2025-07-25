package types

import (
	"testing"
)

func TestMediaType_String(t *testing.T) {
	tests := []struct {
		mediaType MediaType
		expected  string
	}{
		{MediaTypeImage, "images"},
		{MediaTypeDocument, "documents"},
		{MediaTypeVideo, "videos"},
		{MediaTypeAudio, "audio"},
		{MediaTypeOther, "other"},
	}

	for _, test := range tests {
		result := test.mediaType.String()
		if result != test.expected {
			t.Errorf("MediaType.String() = %v, want %v", result, test.expected)
		}
	}
}

func TestParseMediaType(t *testing.T) {
	tests := []struct {
		input    string
		expected MediaType
		hasError bool
	}{
		{"images", MediaTypeImage, false},
		{"image", MediaTypeImage, false},
		{"documents", MediaTypeDocument, false},
		{"document", MediaTypeDocument, false},
		{"docs", MediaTypeDocument, false},
		{"doc", MediaTypeDocument, false},
		{"videos", MediaTypeVideo, false},
		{"video", MediaTypeVideo, false},
		{"audio", MediaTypeAudio, false},
		{"other", MediaTypeOther, false},
		{"unknown", MediaTypeOther, true},
		{"invalid", MediaTypeOther, true},
	}

	for _, test := range tests {
		result, err := ParseMediaType(test.input)

		if test.hasError && err == nil {
			t.Errorf("ParseMediaType(%v) expected error, got none", test.input)
		}

		if !test.hasError && err != nil {
			t.Errorf("ParseMediaType(%v) unexpected error: %v", test.input, err)
		}

		if result != test.expected {
			t.Errorf("ParseMediaType(%v) = %v, want %v", test.input, result, test.expected)
		}
	}
}

func TestGetDefaultMediaConfig(t *testing.T) {
	// Test image config
	imageConfig := GetDefaultMediaConfig(MediaTypeImage)
	if !imageConfig.Enabled {
		t.Error("Image media type should be enabled by default")
	}
	if len(imageConfig.Extensions) == 0 {
		t.Error("Image media type should have extensions defined")
	}
	if imageConfig.S3Prefix != "images/" {
		t.Errorf("Image S3 prefix = %v, want 'images/'", imageConfig.S3Prefix)
	}

	// Test document config
	docConfig := GetDefaultMediaConfig(MediaTypeDocument)
	if docConfig.Enabled {
		t.Error("Document media type should be disabled by default")
	}
	if len(docConfig.Extensions) == 0 {
		t.Error("Document media type should have extensions defined")
	}
	if docConfig.S3Prefix != "documents/" {
		t.Errorf("Document S3 prefix = %v, want 'documents/'", docConfig.S3Prefix)
	}
}

func TestMigrationConfig_GetMediaConfig(t *testing.T) {
	// Test with legacy configuration
	config := &MigrationConfig{
		S3Bucket: "test-bucket",
		S3Prefix: "legacy-prefix/",
	}

	mediaConfig := config.GetMediaConfig(MediaTypeImage)
	if mediaConfig.S3Bucket != "test-bucket" {
		t.Errorf("Expected legacy S3 bucket, got %v", mediaConfig.S3Bucket)
	}
	if mediaConfig.S3Prefix != "legacy-prefix/" {
		t.Errorf("Expected legacy S3 prefix, got %v", mediaConfig.S3Prefix)
	}

	// Test with media-specific configuration
	config.MediaTypes = map[MediaType]*MediaConfig{
		MediaTypeImage: {
			Enabled:    true,
			Extensions: []string{"jpg", "png"},
			S3Bucket:   "image-bucket",
			S3Prefix:   "images/",
		},
	}

	mediaConfig = config.GetMediaConfig(MediaTypeImage)
	if mediaConfig.S3Bucket != "image-bucket" {
		t.Errorf("Expected media-specific S3 bucket, got %v", mediaConfig.S3Bucket)
	}
}

func TestMigrationConfig_IsMediaTypeEnabled(t *testing.T) {
	config := &MigrationConfig{}

	// Test with EnabledMediaTypes specified
	config.EnabledMediaTypes = []MediaType{MediaTypeImage, MediaTypeDocument}

	if !config.IsMediaTypeEnabled(MediaTypeImage) {
		t.Error("Image should be enabled")
	}
	if !config.IsMediaTypeEnabled(MediaTypeDocument) {
		t.Error("Document should be enabled")
	}
	if config.IsMediaTypeEnabled(MediaTypeVideo) {
		t.Error("Video should not be enabled")
	}

	// Test with media-specific configuration
	config.EnabledMediaTypes = nil
	config.MediaTypes = map[MediaType]*MediaConfig{
		MediaTypeImage:    {Enabled: true},
		MediaTypeDocument: {Enabled: false},
	}

	if !config.IsMediaTypeEnabled(MediaTypeImage) {
		t.Error("Image should be enabled via MediaTypes config")
	}
	if config.IsMediaTypeEnabled(MediaTypeDocument) {
		t.Error("Document should not be enabled via MediaTypes config")
	}
}

func TestMigrationConfig_GetEnabledMediaTypes(t *testing.T) {
	config := &MigrationConfig{}

	// Test with explicit EnabledMediaTypes
	config.EnabledMediaTypes = []MediaType{MediaTypeImage, MediaTypeDocument}
	enabled := config.GetEnabledMediaTypes()

	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled media types, got %d", len(enabled))
	}

	// Test default fallback (should return images only)
	config.EnabledMediaTypes = nil
	config.MediaTypes = nil
	enabled = config.GetEnabledMediaTypes()

	if len(enabled) != 1 || enabled[0] != MediaTypeImage {
		t.Errorf("Expected default to be images only, got %v", enabled)
	}
}

func TestMigrationConfig_InitializeMediaTypes(t *testing.T) {
	config := &MigrationConfig{
		S3Bucket: "legacy-bucket",
		S3Prefix: "legacy-prefix/",
	}

	config.InitializeMediaTypes()

	// Check that MediaTypes map is initialized
	if config.MediaTypes == nil {
		t.Error("MediaTypes should be initialized")
	}

	// Check that all media types have configurations
	for mediaType := MediaTypeImage; mediaType <= MediaTypeOther; mediaType++ {
		if _, exists := config.MediaTypes[mediaType]; !exists {
			t.Errorf("MediaType %v should have configuration", mediaType)
		}
	}

	// Check that legacy configuration is applied to images
	imageConfig := config.MediaTypes[MediaTypeImage]
	if imageConfig.S3Bucket != "legacy-bucket" {
		t.Errorf("Legacy bucket not applied to image config, got %v", imageConfig.S3Bucket)
	}
	if imageConfig.S3Prefix != "legacy-prefix/" {
		t.Errorf("Legacy prefix not applied to image config, got %v", imageConfig.S3Prefix)
	}
	if !imageConfig.Enabled {
		t.Error("Image config should be enabled when legacy bucket is present")
	}
}

func TestNewMigrationResult(t *testing.T) {
	result := NewMigrationResult()

	if result.UploadedMedia == nil {
		t.Error("UploadedMedia map should be initialized")
	}
}

func TestMigrationResult_AddUploadedMedia(t *testing.T) {
	result := NewMigrationResult()

	result.AddUploadedMedia(MediaTypeImage, 5)
	result.AddUploadedMedia(MediaTypeDocument, 3)
	result.AddUploadedMedia(MediaTypeImage, 2) // Add more images

	if result.UploadedMedia[MediaTypeImage] != 7 {
		t.Errorf("Expected 7 images, got %d", result.UploadedMedia[MediaTypeImage])
	}
	if result.UploadedMedia[MediaTypeDocument] != 3 {
		t.Errorf("Expected 3 documents, got %d", result.UploadedMedia[MediaTypeDocument])
	}

	// Test backward compatibility
	if result.UploadedImages != 7 {
		t.Errorf("Expected backward compatibility field to be 7, got %d", result.UploadedImages)
	}
}

func TestMigrationResult_GetTotalUploaded(t *testing.T) {
	result := NewMigrationResult()

	result.AddUploadedMedia(MediaTypeImage, 5)
	result.AddUploadedMedia(MediaTypeDocument, 3)
	result.AddUploadedMedia(MediaTypeVideo, 2)

	total := result.GetTotalUploaded()
	if total != 10 {
		t.Errorf("Expected total of 10, got %d", total)
	}
}
