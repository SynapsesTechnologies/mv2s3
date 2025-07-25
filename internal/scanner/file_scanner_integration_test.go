package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// Test ScanForMediaReferences
func TestFileScanner_ScanForMediaReferences(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "mv2s3-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file with media references
	testFile := filepath.Join(tempDir, "test.md")
	content := `# Test Document

![Image](./images/logo.png)
[Manual](./docs/manual.pdf)
[Spreadsheet](./files/data.xlsx)
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Configure scanner with all media types enabled
	config := &types.MigrationConfig{
		SourceDir:      tempDir,
		FileExtensions: []string{"md", "html"},
		MediaTypes: map[types.MediaType]*types.MediaConfig{
			types.MediaTypeImage:    {Enabled: true},
			types.MediaTypeDocument: {Enabled: true},
			types.MediaTypeOther:    {Enabled: true},
		},
	}

	scanner := NewFileScanner(config)
	references, err := scanner.ScanForMediaReferences()

	if err != nil {
		t.Fatalf("ScanForMediaReferences failed: %v", err)
	}

	if len(references) != 3 {
		t.Errorf("Expected 3 references, got %d", len(references))
	}

	// Verify reference types
	expectedTypes := map[types.MediaType]int{
		types.MediaTypeImage:    1,
		types.MediaTypeDocument: 2, // manual.pdf and data.xlsx are both documents
	}

	actualTypes := make(map[types.MediaType]int)
	for _, ref := range references {
		actualTypes[ref.MediaType]++
	}

	for expectedType, expectedCount := range expectedTypes {
		if actualTypes[expectedType] != expectedCount {
			t.Errorf("Expected %d references of type %s, got %d",
				expectedCount, expectedType, actualTypes[expectedType])
		}
	}
}

// Test ScanMediaFiles
func TestFileScanner_ScanMediaFiles(t *testing.T) {
	// Create temporary test directory structure
	tempDir, err := os.MkdirTemp("", "mv2s3-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	imgDir := filepath.Join(tempDir, "images")
	docDir := filepath.Join(tempDir, "docs")
	os.MkdirAll(imgDir, 0755)
	os.MkdirAll(docDir, 0755)

	// Create test media files
	testFiles := []struct {
		path  string
		type_ types.MediaType
	}{
		{filepath.Join(imgDir, "logo.png"), types.MediaTypeImage},
		{filepath.Join(imgDir, "banner.jpg"), types.MediaTypeImage},
		{filepath.Join(docDir, "manual.pdf"), types.MediaTypeDocument},
		{filepath.Join(docDir, "guide.docx"), types.MediaTypeDocument},
		{filepath.Join(tempDir, "readme.txt"), types.MediaTypeDocument},
	}

	for _, tf := range testFiles {
		err = os.WriteFile(tf.path, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.path, err)
		}
	}

	// Configure scanner
	config := &types.MigrationConfig{
		SourceDir: tempDir,
		MediaTypes: map[types.MediaType]*types.MediaConfig{
			types.MediaTypeImage:    {Enabled: true},
			types.MediaTypeDocument: {Enabled: true},
		},
	}

	scanner := NewFileScanner(config)
	mediaFiles, err := scanner.ScanMediaFiles()

	if err != nil {
		t.Fatalf("ScanMediaFiles failed: %v", err)
	}

	// Check image files
	if len(mediaFiles[types.MediaTypeImage]) != 2 {
		t.Errorf("Expected 2 image files, got %d", len(mediaFiles[types.MediaTypeImage]))
	}

	// Check document files
	if len(mediaFiles[types.MediaTypeDocument]) != 3 {
		t.Errorf("Expected 3 document files, got %d", len(mediaFiles[types.MediaTypeDocument]))
	}
}

// Test IsValidFileForMediaType
func TestFileScanner_IsValidFileForMediaType(t *testing.T) {
	config := &types.MigrationConfig{
		MediaTypes: map[types.MediaType]*types.MediaConfig{
			types.MediaTypeImage:    {Enabled: true},
			types.MediaTypeDocument: {Enabled: true},
			types.MediaTypeVideo:    {Enabled: false},
		},
	}

	scanner := NewFileScanner(config)

	testCases := []struct {
		filePath  string
		mediaType types.MediaType
		expected  bool
	}{
		{"test.jpg", types.MediaTypeImage, true},
		{"test.png", types.MediaTypeImage, true},
		{"test.pdf", types.MediaTypeDocument, true},
		{"test.docx", types.MediaTypeDocument, true},
		{"test.mp4", types.MediaTypeVideo, false},     // Video disabled
		{"test.mp4", types.MediaTypeImage, false},     // Wrong type
		{"test.unknown", types.MediaTypeOther, false}, // Other disabled
	}

	for _, tc := range testCases {
		t.Run(tc.filePath, func(t *testing.T) {
			result := scanner.IsValidFileForMediaType(tc.filePath, tc.mediaType)
			if result != tc.expected {
				t.Errorf("IsValidFileForMediaType(%s, %s) = %v, expected %v",
					tc.filePath, tc.mediaType, result, tc.expected)
			}
		})
	}
}

// Test GetSupportedExtensions
func TestFileScanner_GetSupportedExtensions(t *testing.T) {
	config := &types.MigrationConfig{
		MediaTypes: map[types.MediaType]*types.MediaConfig{
			types.MediaTypeImage: {
				Enabled:    true,
				Extensions: []string{"jpg", "png", "gif"},
			},
			types.MediaTypeDocument: {
				Enabled:    true,
				Extensions: []string{"pdf", "docx"},
			},
			types.MediaTypeVideo: {
				Enabled:    false,
				Extensions: []string{"mp4", "avi"},
			},
		},
	}

	scanner := NewFileScanner(config)
	extensions := scanner.GetSupportedExtensions()

	expectedExtensions := []string{"jpg", "png", "gif", "pdf", "docx"}

	if len(extensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(extensions))
	}

	// Convert to map for easier checking
	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[ext] = true
	}

	for _, expected := range expectedExtensions {
		if !extMap[expected] {
			t.Errorf("Expected extension %s not found in result", expected)
		}
	}

	// Ensure disabled video extensions are not included
	for _, ext := range []string{"mp4", "avi"} {
		if extMap[ext] {
			t.Errorf("Disabled extension %s should not be included", ext)
		}
	}
}

// Test backward compatibility with ScanForImageReferences
func TestFileScanner_ScanForImageReferences_BackwardCompatibility(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "mv2s3-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test markdown file with image references
	testFile := filepath.Join(tempDir, "test.md")
	content := `# Test Document

![Image](./images/logo.png)
![Banner](./images/banner.jpg)
[Document](./docs/manual.pdf)  <!-- This should be ignored by image scanner -->
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Configure scanner
	config := &types.MigrationConfig{
		SourceDir:      tempDir,
		FileExtensions: []string{"md"},
	}

	scanner := NewFileScanner(config)
	references, err := scanner.ScanForImageReferences()

	if err != nil {
		t.Fatalf("ScanForImageReferences failed: %v", err)
	}

	if len(references) != 2 {
		t.Errorf("Expected 2 image references, got %d", len(references))
	}

	// Verify all references are images
	for _, ref := range references {
		if !strings.HasSuffix(ref.LocalPath, ".png") && !strings.HasSuffix(ref.LocalPath, ".jpg") {
			t.Errorf("Expected image file, got %s", ref.LocalPath)
		}
	}
}
