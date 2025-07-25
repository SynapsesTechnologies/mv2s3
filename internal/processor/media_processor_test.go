package processor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// Test NewMediaProcessor
func TestNewMediaProcessor(t *testing.T) {
	processor := NewMediaProcessor()
	if processor == nil {
		t.Fatal("NewMediaProcessor returned nil")
		return
	}

	if processor.imageProcessor == nil {
		t.Error("MediaProcessor should have an imageProcessor")
	}
}

// Test ValidateMediaFile
func TestMediaProcessor_ValidateMediaFile(t *testing.T) {
	processor := NewMediaProcessor()

	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "media-processor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	testCases := []struct {
		name      string
		fileName  string
		content   string
		mediaType types.MediaType
		expectErr bool
	}{
		{
			name:      "Valid PDF file",
			fileName:  "test.pdf",
			content:   "%PDF-1.4\nSample PDF content",
			mediaType: types.MediaTypeDocument,
			expectErr: false,
		},
		{
			name:      "Invalid PDF file",
			fileName:  "bad.pdf",
			content:   "Not a PDF file",
			mediaType: types.MediaTypeDocument,
			expectErr: true,
		},
		{
			name:      "Valid text file",
			fileName:  "test.txt",
			content:   "Sample text content",
			mediaType: types.MediaTypeDocument,
			expectErr: false,
		},
		{
			name:      "Valid video file",
			fileName:  "test.mp4",
			content:   "Sample video content",
			mediaType: types.MediaTypeVideo,
			expectErr: false,
		},
		{
			name:      "Empty file",
			fileName:  "empty.pdf",
			content:   "",
			mediaType: types.MediaTypeDocument,
			expectErr: true,
		},
		{
			name:      "Non-existent file",
			fileName:  "", // Will not be created
			content:   "",
			mediaType: types.MediaTypeDocument,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var filePath string

			if tc.fileName != "" {
				filePath = filepath.Join(tempDir, tc.fileName)
				err = os.WriteFile(filePath, []byte(tc.content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			} else {
				filePath = filepath.Join(tempDir, "nonexistent.pdf")
			}

			err = processor.ValidateMediaFile(filePath, tc.mediaType)

			if tc.expectErr && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tc.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test GetMediaInfo
func TestMediaProcessor_GetMediaInfo(t *testing.T) {
	processor := NewMediaProcessor()

	// Create temporary test file
	tempDir, err := os.MkdirTemp("", "media-processor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.jpg")
	testContent := "fake image content"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	info, err := processor.GetMediaInfo(testFile, types.MediaTypeImage)
	if err != nil {
		t.Fatalf("GetMediaInfo failed: %v", err)
	}

	if info.FilePath != testFile {
		t.Errorf("Expected file path %s, got %s", testFile, info.FilePath)
	}

	if info.MediaType != types.MediaTypeImage {
		t.Errorf("Expected media type %s, got %s", types.MediaTypeImage, info.MediaType)
	}

	if info.FileSize != int64(len(testContent)) {
		t.Errorf("Expected file size %d, got %d", len(testContent), info.FileSize)
	}

	if info.ContentType != "image/jpeg" {
		t.Errorf("Expected content type image/jpeg, got %s", info.ContentType)
	}

	// Check that LastModified is recent (within last minute)
	if time.Since(info.LastModified) > time.Minute {
		t.Errorf("LastModified time seems too old: %v", info.LastModified)
	}
}

// Test getContentType
func TestMediaProcessor_getContentType(t *testing.T) {
	processor := NewMediaProcessor()

	testCases := []struct {
		filePath     string
		mediaType    types.MediaType
		expectedMIME string
	}{
		// Images
		{"image.jpg", types.MediaTypeImage, "image/jpeg"},
		{"image.jpeg", types.MediaTypeImage, "image/jpeg"},
		{"image.png", types.MediaTypeImage, "image/png"},
		{"image.gif", types.MediaTypeImage, "image/gif"},
		{"image.svg", types.MediaTypeImage, "image/svg+xml"},
		{"image.webp", types.MediaTypeImage, "image/webp"},

		// Documents
		{"doc.pdf", types.MediaTypeDocument, "application/pdf"},
		{"doc.docx", types.MediaTypeDocument, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"doc.txt", types.MediaTypeDocument, "text/plain"},
		{"doc.rtf", types.MediaTypeDocument, "application/rtf"},

		// Videos
		{"video.mp4", types.MediaTypeVideo, "video/mp4"},
		{"video.avi", types.MediaTypeVideo, "video/x-msvideo"},
		{"video.mov", types.MediaTypeVideo, "video/quicktime"},

		// Audio
		{"audio.mp3", types.MediaTypeAudio, "audio/mpeg"},
		{"audio.wav", types.MediaTypeAudio, "audio/wav"},
		{"audio.ogg", types.MediaTypeAudio, "audio/ogg"},

		// Unknown extensions
		{"file.unknown", types.MediaTypeImage, "image/jpeg"},
		{"file.unknown", types.MediaTypeDocument, "application/octet-stream"},
		{"file.unknown", types.MediaTypeVideo, "video/mp4"},
		{"file.unknown", types.MediaTypeAudio, "audio/mpeg"},
		{"file.unknown", types.MediaTypeOther, "application/octet-stream"},
	}

	for _, tc := range testCases {
		t.Run(tc.filePath, func(t *testing.T) {
			result := processor.getContentType(tc.filePath, tc.mediaType)
			if result != tc.expectedMIME {
				t.Errorf("Expected MIME type %s, got %s", tc.expectedMIME, result)
			}
		})
	}
}

// Test validatePDFFile
func TestMediaProcessor_validatePDFFile(t *testing.T) {
	processor := NewMediaProcessor()

	tempDir, err := os.MkdirTemp("", "pdf-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test valid PDF
	validPDF := filepath.Join(tempDir, "valid.pdf")
	err = os.WriteFile(validPDF, []byte("%PDF-1.4\nSample content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = processor.validatePDFFile(validPDF)
	if err != nil {
		t.Errorf("Valid PDF should not error: %v", err)
	}

	// Test invalid PDF
	invalidPDF := filepath.Join(tempDir, "invalid.pdf")
	err = os.WriteFile(invalidPDF, []byte("Not a PDF"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = processor.validatePDFFile(invalidPDF)
	if err == nil {
		t.Error("Invalid PDF should error")
	}
}

// Test MediaInfo struct
func TestMediaInfo(t *testing.T) {
	now := time.Now()
	info := &MediaInfo{
		FilePath:     "/test/file.jpg",
		MediaType:    types.MediaTypeImage,
		FileSize:     1024,
		LastModified: now,
		ContentType:  "image/jpeg",
	}

	if info.FilePath != "/test/file.jpg" {
		t.Errorf("FilePath not set correctly")
	}

	if info.MediaType != types.MediaTypeImage {
		t.Errorf("MediaType not set correctly")
	}

	if info.FileSize != 1024 {
		t.Errorf("FileSize not set correctly")
	}

	if info.LastModified != now {
		t.Errorf("LastModified not set correctly")
	}

	if info.ContentType != "image/jpeg" {
		t.Errorf("ContentType not set correctly")
	}
}
