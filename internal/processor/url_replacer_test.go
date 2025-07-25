package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestNewURLReplacer(t *testing.T) {
	replacer := NewURLReplacer(false, false, false)
	if replacer == nil {
		t.Error("Expected NewURLReplacer to return a non-nil replacer")
	}
}

func TestURLReplacer_ReplaceURLsInFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	originalContent := `# Test Document

This is a test with images:
![Logo](images/logo.png)
![Screenshot](./images/screenshot.png)

And some HTML:
<img src="images/header.jpg" alt="Header">
<img src="./images/footer.svg" alt="Footer" />
`

	expectedContent := `# Test Document

This is a test with images:
![Logo](https://bucket.s3.region.amazonaws.com/logo.png)
![Screenshot](https://bucket.s3.region.amazonaws.com/screenshot.png)

And some HTML:
<img src="https://bucket.s3.region.amazonaws.com/header.jpg" alt="Header">
<img src="https://bucket.s3.region.amazonaws.com/footer.svg" alt="Footer" />
`

	// Write test file
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create URL replacer
	replacer := NewURLReplacer(false, false, false)

	// Define replacements
	replacements := map[string]string{
		"images/logo.png":         "https://bucket.s3.region.amazonaws.com/logo.png",
		"./images/screenshot.png": "https://bucket.s3.region.amazonaws.com/screenshot.png",
		"images/header.jpg":       "https://bucket.s3.region.amazonaws.com/header.jpg",
		"./images/footer.svg":     "https://bucket.s3.region.amazonaws.com/footer.svg",
	}

	// Apply replacements
	err := replacer.ReplaceURLsInFile(testFile, replacements)
	if err != nil {
		t.Fatalf("ReplaceURLsInFile failed: %v", err)
	}

	// Read result
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	if string(result) != expectedContent {
		t.Errorf("Content mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, string(result))
	}
}

func TestURLReplacer_UpdateImageReferences(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	// Create test markdown file
	mdFile := filepath.Join(tmpDir, "test.md")
	mdContent := `# Test
![Logo](images/logo.png)
![Icon](./icons/icon.svg)
`
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to create markdown file: %v", err)
	}

	// Create test HTML file
	htmlFile := filepath.Join(tmpDir, "test.html")
	htmlContent := `<html>
<body>
<img src="images/header.jpg" alt="Header">
</body>
</html>
`
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}

	// Create image references
	refs := []types.ImageReference{
		{
			SourceFile:  mdFile,
			OriginalURL: "images/logo.png",
			LineNumber:  2,
		},
		{
			SourceFile:  mdFile,
			OriginalURL: "./icons/icon.svg",
			LineNumber:  3,
		},
		{
			SourceFile:  htmlFile,
			OriginalURL: "images/header.jpg",
			LineNumber:  3,
		},
	}

	// Create URL mapping
	urlMapping := map[string]string{
		"images/logo.png":   "https://bucket.s3.region.amazonaws.com/logo.png",
		"./icons/icon.svg":  "https://bucket.s3.region.amazonaws.com/icon.svg",
		"images/header.jpg": "https://bucket.s3.region.amazonaws.com/header.jpg",
	}

	// Create URL replacer
	replacer := NewURLReplacer(false, false, false)

	// Update references
	err := replacer.UpdateImageReferences(refs, urlMapping)
	if err != nil {
		t.Fatalf("UpdateImageReferences failed: %v", err)
	}

	// Check markdown file
	mdResult, err := os.ReadFile(mdFile)
	if err != nil {
		t.Fatalf("Failed to read markdown result: %v", err)
	}

	if !strings.Contains(string(mdResult), "https://bucket.s3.region.amazonaws.com/logo.png") {
		t.Errorf("Markdown file doesn't contain expected S3 URL for logo")
	}
	if !strings.Contains(string(mdResult), "https://bucket.s3.region.amazonaws.com/icon.svg") {
		t.Errorf("Markdown file doesn't contain expected S3 URL for icon")
	}

	// Check HTML file
	htmlResult, err := os.ReadFile(htmlFile)
	if err != nil {
		t.Fatalf("Failed to read HTML result: %v", err)
	}

	if !strings.Contains(string(htmlResult), "https://bucket.s3.region.amazonaws.com/header.jpg") {
		t.Errorf("HTML file doesn't contain expected S3 URL for header")
	}
}

func TestURLReplacer_DryRun(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	originalContent := `![Logo](images/logo.png)`

	// Write test file
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create URL replacer in dry-run mode
	replacer := NewURLReplacer(false, true, false)

	// Define replacements
	replacements := map[string]string{
		"images/logo.png": "https://bucket.s3.region.amazonaws.com/logo.png",
	}

	// Apply replacements (should not modify file in dry-run)
	err := replacer.ReplaceURLsInFile(testFile, replacements)
	if err != nil {
		t.Fatalf("ReplaceURLsInFile failed: %v", err)
	}

	// Read result - should be unchanged
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	if string(result) != originalContent {
		t.Errorf("File was modified during dry-run. Expected: %s, Got: %s", originalContent, string(result))
	}
}
