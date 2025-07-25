package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// TestProcessorIntegration tests the complete processor pipeline
func TestProcessorIntegration(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "processor-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test media files
	mediaFiles := map[string][]byte{
		"doc.pdf":    []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n>>\nendobj"),
		"image.jpg":  []byte("\xff\xd8\xff\xe0\x00\x10JFIF"),
		"video.mp4":  []byte("ftypmp4"),
		"data.xlsx":  []byte("PK\x03\x04"), // ZIP signature for Office files
		"audio.mp3":  []byte("ID3"),
		"readme.txt": []byte("This is a text file"),
	}

	for filename, content := range mediaFiles {
		filepath := filepath.Join(tempDir, filename)
		err = os.WriteFile(filepath, content, 0644)
		if err != nil {
			t.Fatalf("Failed to create media file %s: %v", filename, err)
		}
	}

	// Create test source file with various media references
	sourceFile := filepath.Join(tempDir, "index.html")
	sourceContent := `<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
    <style>
        .hero { background: url('./image.jpg') no-repeat; }
    </style>
</head>
<body>
    <h1>Media Integration Test</h1>
    
    <!-- Images -->
    <img src="./image.jpg" alt="Test Image">
    
    <!-- Documents -->
    <a href="./doc.pdf">Download PDF</a>
    <a href="./data.xlsx">Download Excel</a>
    <a href="./readme.txt">Read Text</a>
    
    <!-- Media -->
    <video src="./video.mp4" controls></video>
    <audio src="./audio.mp3" controls></audio>
    
    <!-- Embed -->
    <embed src="./doc.pdf" type="application/pdf">
    <object data="./data.xlsx" type="application/vnd.ms-excel"></object>
</body>
</html>`

	err = os.WriteFile(sourceFile, []byte(sourceContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create media references
	mediaRefs := []types.MediaReference{
		{
			SourceFile:  sourceFile,
			LocalPath:   filepath.Join(tempDir, "doc.pdf"),
			OriginalURL: "./doc.pdf",
			MediaType:   types.MediaTypeDocument,
		},
		{
			SourceFile:  sourceFile,
			LocalPath:   filepath.Join(tempDir, "image.jpg"),
			OriginalURL: "./image.jpg",
			MediaType:   types.MediaTypeImage,
		},
		{
			SourceFile:  sourceFile,
			LocalPath:   filepath.Join(tempDir, "video.mp4"),
			OriginalURL: "./video.mp4",
			MediaType:   types.MediaTypeVideo,
		},
		{
			SourceFile:  sourceFile,
			LocalPath:   filepath.Join(tempDir, "data.xlsx"),
			OriginalURL: "./data.xlsx",
			MediaType:   types.MediaTypeDocument,
		},
		{
			SourceFile:  sourceFile,
			LocalPath:   filepath.Join(tempDir, "audio.mp3"),
			OriginalURL: "./audio.mp3",
			MediaType:   types.MediaTypeAudio,
		},
		{
			SourceFile:  sourceFile,
			LocalPath:   filepath.Join(tempDir, "readme.txt"),
			OriginalURL: "./readme.txt",
			MediaType:   types.MediaTypeDocument,
		},
	}

	// Step 1: Validate media files using MediaProcessor
	t.Run("MediaProcessor Validation", func(t *testing.T) {
		mediaProcessor := NewMediaProcessor()

		for _, ref := range mediaRefs {
			err := mediaProcessor.ValidateMediaFile(ref.LocalPath, ref.MediaType)
			if err != nil {
				t.Errorf("Failed to validate %s (%s): %v", ref.LocalPath, ref.MediaType, err)
			}

			// Get media info
			info, err := mediaProcessor.GetMediaInfo(ref.LocalPath, ref.MediaType)
			if err != nil {
				t.Errorf("Failed to get info for %s: %v", ref.LocalPath, err)
			}

			// Verify basic info
			if info.FileSize == 0 {
				t.Errorf("Expected non-zero size for %s", ref.LocalPath)
			}
			if info.LastModified.IsZero() {
				t.Errorf("Expected valid mod time for %s", ref.LocalPath)
			}
		}
	})

	// Step 2: Generate S3 URLs (simulated)
	urlMapping := map[string]string{
		"./doc.pdf":    "https://s3.amazonaws.com/bucket/documents/doc.pdf",
		"./image.jpg":  "https://s3.amazonaws.com/bucket/images/image.jpg",
		"./video.mp4":  "https://s3.amazonaws.com/bucket/videos/video.mp4",
		"./data.xlsx":  "https://s3.amazonaws.com/bucket/documents/data.xlsx",
		"./audio.mp3":  "https://s3.amazonaws.com/bucket/audio/audio.mp3",
		"./readme.txt": "https://s3.amazonaws.com/bucket/documents/readme.txt",
	}

	// Step 3: Update URLs using URLReplacer
	t.Run("URLReplacer Integration", func(t *testing.T) {
		replacer := NewURLReplacer(false, false, false)

		// Validate file access
		err := replacer.ValidateMediaFileAccess(mediaRefs)
		if err != nil {
			t.Fatalf("Failed to validate file access: %v", err)
		}

		// Update references
		err = replacer.UpdateMediaReferences(mediaRefs, urlMapping)
		if err != nil {
			t.Fatalf("Failed to update media references: %v", err)
		}

		// Read updated content
		updatedBytes, err := os.ReadFile(sourceFile)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		updatedContent := string(updatedBytes)

		// Verify all URLs were replaced
		for originalURL, newURL := range urlMapping {
			if strings.Contains(updatedContent, originalURL) {
				t.Errorf("Original URL %s still found in content", originalURL)
			}
			if !strings.Contains(updatedContent, newURL) {
				t.Errorf("New URL %s not found in content", newURL)
			}
		}

		// Verify specific replacements
		expectedPatterns := []string{
			`<img src="https://s3.amazonaws.com/bucket/images/image.jpg"`,
			`<a href="https://s3.amazonaws.com/bucket/documents/doc.pdf"`,
			`<a href="https://s3.amazonaws.com/bucket/documents/data.xlsx"`,
			`<a href="https://s3.amazonaws.com/bucket/documents/readme.txt"`,
			`<video src="https://s3.amazonaws.com/bucket/videos/video.mp4"`,
			`<audio src="https://s3.amazonaws.com/bucket/audio/audio.mp3"`,
			`<embed src="https://s3.amazonaws.com/bucket/documents/doc.pdf"`,
			`<object data="https://s3.amazonaws.com/bucket/documents/data.xlsx"`,
			`background: url('https://s3.amazonaws.com/bucket/images/image.jpg')`,
		}

		for _, pattern := range expectedPatterns {
			if !strings.Contains(updatedContent, pattern) {
				t.Errorf("Expected pattern not found: %s", pattern)
			}
		}
	})

	// Step 4: Verify processing statistics
	t.Run("Processing Statistics", func(t *testing.T) {
		// Count references by type
		typeCount := make(map[types.MediaType]int)
		for _, ref := range mediaRefs {
			typeCount[ref.MediaType]++
		}

		expectedCounts := map[types.MediaType]int{
			types.MediaTypeDocument: 3, // PDF, XLSX, TXT
			types.MediaTypeImage:    1, // JPG
			types.MediaTypeVideo:    1, // MP4
			types.MediaTypeAudio:    1, // MP3
		}

		for mediaType, expectedCount := range expectedCounts {
			if typeCount[mediaType] != expectedCount {
				t.Errorf("Expected %d %s files, got %d", expectedCount, mediaType, typeCount[mediaType])
			}
		}

		// Total references
		if len(mediaRefs) != 6 {
			t.Errorf("Expected 6 total references, got %d", len(mediaRefs))
		}
	})
}

// TestProcessorError tests error handling in the processor pipeline
func TestProcessorError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "processor-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("MediaProcessor Invalid File", func(t *testing.T) {
		processor := NewMediaProcessor()

		// Test non-existent file
		err := processor.ValidateMediaFile("/nonexistent/file.pdf", types.MediaTypeDocument)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}

		// Test invalid PDF
		invalidPDF := filepath.Join(tempDir, "invalid.pdf")
		err = os.WriteFile(invalidPDF, []byte("not a pdf"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid PDF: %v", err)
		}

		err = processor.ValidateMediaFile(invalidPDF, types.MediaTypeDocument)
		if err == nil {
			t.Error("Expected error for invalid PDF")
		}
	})

	t.Run("URLReplacer File Access Error", func(t *testing.T) {
		replacer := NewURLReplacer(false, false, false)

		// Test with non-existent source file
		refs := []types.MediaReference{
			{
				SourceFile:  "/nonexistent/file.md",
				LocalPath:   "./image.jpg",
				OriginalURL: "./image.jpg",
				MediaType:   types.MediaTypeImage,
			},
		}

		err := replacer.ValidateMediaFileAccess(refs)
		if err == nil {
			t.Error("Expected error for non-existent source file")
		}
	})
}

// TestProcessorDryRun tests dry run functionality
func TestProcessorDryRun(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "processor-dryrun-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.md")
	originalContent := `![Image](./test.jpg)
[Document](./test.pdf)`

	err = os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create read-only file to test dry run doesn't fail on permission issues
	readOnlyFile := filepath.Join(tempDir, "readonly.md")
	err = os.WriteFile(readOnlyFile, []byte(originalContent), 0444)
	if err != nil {
		t.Fatalf("Failed to create read-only file: %v", err)
	}

	refs := []types.MediaReference{
		{
			SourceFile:  readOnlyFile,
			LocalPath:   "./test.jpg",
			OriginalURL: "./test.jpg",
			MediaType:   types.MediaTypeImage,
		},
	}

	urlMapping := map[string]string{
		"./test.jpg": "https://s3.example.com/test.jpg",
	}

	// Test dry run mode
	dryRunReplacer := NewURLReplacer(false, true, false)
	err = dryRunReplacer.UpdateMediaReferences(refs, urlMapping)
	if err != nil {
		t.Fatalf("Dry run should not fail on read-only files: %v", err)
	}

	// Verify file was not actually modified
	content, err := os.ReadFile(readOnlyFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != originalContent {
		t.Error("File should not be modified in dry run mode")
	}
}
