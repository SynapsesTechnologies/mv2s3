package scanner

import (
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestNewMediaTypeDetector(t *testing.T) {
	detector := NewMediaTypeDetector()

	if detector == nil {
		t.Error("NewMediaTypeDetector should not return nil")
	}

	if detector.extensionCache == nil {
		t.Error("Extension cache should be initialized")
	}

	// Check that some known extensions are cached
	if detector.extensionCache["jpg"] != types.MediaTypeImage {
		t.Error("JPG should be mapped to image type")
	}
	if detector.extensionCache["pdf"] != types.MediaTypeDocument {
		t.Error("PDF should be mapped to document type")
	}
}

func TestMediaTypeDetector_DetectMediaType(t *testing.T) {
	detector := NewMediaTypeDetector()

	tests := []struct {
		filePath string
		expected types.MediaType
	}{
		{"image.jpg", types.MediaTypeImage},
		{"image.JPG", types.MediaTypeImage}, // Test case insensitivity
		{"document.pdf", types.MediaTypeDocument},
		{"document.PDF", types.MediaTypeDocument},
		{"video.mp4", types.MediaTypeVideo},
		{"audio.mp3", types.MediaTypeAudio},
		{"unknown.xyz", types.MediaTypeOther},
		{"no-extension", types.MediaTypeOther},
		{"/path/to/image.png", types.MediaTypeImage},
		{"/path/to/document.docx", types.MediaTypeDocument},
	}

	for _, test := range tests {
		result := detector.DetectMediaType(test.filePath)
		if result != test.expected {
			t.Errorf("DetectMediaType(%v) = %v, want %v", test.filePath, result, test.expected)
		}
	}
}

func TestMediaTypeDetector_IsMediaTypeEnabled(t *testing.T) {
	detector := NewMediaTypeDetector()

	config := &types.MigrationConfig{
		EnabledMediaTypes: []types.MediaType{types.MediaTypeImage, types.MediaTypeDocument},
	}

	if !detector.IsMediaTypeEnabled(types.MediaTypeImage, config) {
		t.Error("Image should be enabled")
	}
	if !detector.IsMediaTypeEnabled(types.MediaTypeDocument, config) {
		t.Error("Document should be enabled")
	}
	if detector.IsMediaTypeEnabled(types.MediaTypeVideo, config) {
		t.Error("Video should not be enabled")
	}
}

func TestMediaTypeDetector_IsFileSupported(t *testing.T) {
	detector := NewMediaTypeDetector()

	config := &types.MigrationConfig{
		EnabledMediaTypes: []types.MediaType{types.MediaTypeImage},
	}

	if !detector.IsFileSupported("image.jpg", config) {
		t.Error("JPG image should be supported")
	}
	if detector.IsFileSupported("document.pdf", config) {
		t.Error("PDF document should not be supported when documents are disabled")
	}
}

func TestMediaTypeDetector_GetSupportedExtensions(t *testing.T) {
	detector := NewMediaTypeDetector()

	config := &types.MigrationConfig{}
	config.MediaTypes = map[types.MediaType]*types.MediaConfig{
		types.MediaTypeImage: {
			Enabled:    true,
			Extensions: []string{"jpg", "png"},
		},
		types.MediaTypeDocument: {
			Enabled:    true,
			Extensions: []string{"pdf", "doc"},
		},
	}
	config.EnabledMediaTypes = []types.MediaType{types.MediaTypeImage, types.MediaTypeDocument}

	extensions := detector.GetSupportedExtensions(config)

	// Should contain extensions from both enabled media types
	expectedExtensions := map[string]bool{
		"jpg": true,
		"png": true,
		"pdf": true,
		"doc": true,
	}

	if len(extensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(extensions))
	}

	for _, ext := range extensions {
		if !expectedExtensions[ext] {
			t.Errorf("Unexpected extension: %v", ext)
		}
	}
}

func TestMediaTypeDetector_GetMediaTypeStats(t *testing.T) {
	detector := NewMediaTypeDetector()

	config := &types.MigrationConfig{
		EnabledMediaTypes: []types.MediaType{types.MediaTypeImage, types.MediaTypeDocument},
	}

	filePaths := []string{
		"image1.jpg",
		"image2.png",
		"document1.pdf",
		"video1.mp4", // Should be excluded
		"document2.doc",
	}

	stats := detector.GetMediaTypeStats(filePaths, config)

	if stats[types.MediaTypeImage] != 2 {
		t.Errorf("Expected 2 images, got %d", stats[types.MediaTypeImage])
	}
	if stats[types.MediaTypeDocument] != 2 {
		t.Errorf("Expected 2 documents, got %d", stats[types.MediaTypeDocument])
	}
	if stats[types.MediaTypeVideo] != 0 {
		t.Errorf("Expected 0 videos (disabled), got %d", stats[types.MediaTypeVideo])
	}
}

func TestMediaTypeDetector_FilterFilesByMediaType(t *testing.T) {
	detector := NewMediaTypeDetector()

	filePaths := []string{
		"image1.jpg",
		"document1.pdf",
		"image2.png",
		"video1.mp4",
		"document2.doc",
	}

	imageFiles := detector.FilterFilesByMediaType(filePaths, types.MediaTypeImage)
	expectedImages := []string{"image1.jpg", "image2.png"}

	if len(imageFiles) != len(expectedImages) {
		t.Errorf("Expected %d image files, got %d", len(expectedImages), len(imageFiles))
	}

	for i, file := range imageFiles {
		if file != expectedImages[i] {
			t.Errorf("Expected %v, got %v", expectedImages[i], file)
		}
	}

	documentFiles := detector.FilterFilesByMediaType(filePaths, types.MediaTypeDocument)
	expectedDocs := []string{"document1.pdf", "document2.doc"}

	if len(documentFiles) != len(expectedDocs) {
		t.Errorf("Expected %d document files, got %d", len(expectedDocs), len(documentFiles))
	}
}

func TestMediaTypeDetector_UpdateExtensionCache(t *testing.T) {
	detector := NewMediaTypeDetector()

	// Add custom extension mapping
	customMappings := map[string]types.MediaType{
		".custom": types.MediaTypeOther,
		"special": types.MediaTypeDocument, // Test without dot prefix
	}

	detector.UpdateExtensionCache(customMappings)

	// Test that custom mappings work
	if detector.DetectMediaType("file.custom") != types.MediaTypeOther {
		t.Error("Custom extension should map to Other type")
	}
	if detector.DetectMediaType("file.special") != types.MediaTypeDocument {
		t.Error("Special extension should map to Document type")
	}
}

func TestMediaTypeDetector_GetExtensionMapping(t *testing.T) {
	detector := NewMediaTypeDetector()

	mapping := detector.GetExtensionMapping()

	// Should return a copy, not the original
	if &mapping == &detector.extensionCache {
		t.Error("Should return a copy of extension mapping, not original")
	}

	// Should contain expected mappings
	if mapping["jpg"] != types.MediaTypeImage {
		t.Error("Mapping should contain jpg -> image")
	}
	if mapping["pdf"] != types.MediaTypeDocument {
		t.Error("Mapping should contain pdf -> document")
	}

	// Modifying returned mapping should not affect original
	originalJpgType := detector.extensionCache["jpg"]
	mapping["jpg"] = types.MediaTypeOther

	if detector.extensionCache["jpg"] != originalJpgType {
		t.Error("Modifying returned mapping should not affect original cache")
	}
}
