package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// Test UpdateMediaReferences
func TestURLReplacer_UpdateMediaReferences(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "url-replacer-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file with various media references
	testFile := filepath.Join(tempDir, "test.md")
	originalContent := `# Test Document

![Image](./images/logo.png)
[Manual](./docs/manual.pdf)
[Video](./videos/demo.mp4)

<img src="./images/banner.jpg" alt="Banner">
<a href="./docs/guide.docx">Download Guide</a>
<embed src="./presentations/slide.pptx">

---
image: ./images/hero.jpg
document: ./docs/readme.txt
---

{{< image src="./images/gallery.png" >}}
{{< download file="./docs/spec.pdf" >}}

.hero {
    background: url('./images/bg.jpg') no-repeat;
}
`

	err = os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create media references
	refs := []types.MediaReference{
		{
			SourceFile:  testFile,
			LocalPath:   "./images/logo.png",
			OriginalURL: "./images/logo.png",
			MediaType:   types.MediaTypeImage,
		},
		{
			SourceFile:  testFile,
			LocalPath:   "./docs/manual.pdf",
			OriginalURL: "./docs/manual.pdf",
			MediaType:   types.MediaTypeDocument,
		},
		{
			SourceFile:  testFile,
			LocalPath:   "./images/banner.jpg",
			OriginalURL: "./images/banner.jpg",
			MediaType:   types.MediaTypeImage,
		},
		{
			SourceFile:  testFile,
			LocalPath:   "./docs/guide.docx",
			OriginalURL: "./docs/guide.docx",
			MediaType:   types.MediaTypeDocument,
		},
	}

	// Create URL mapping
	urlMapping := map[string]string{
		"./images/logo.png":   "https://s3.amazonaws.com/bucket/images/logo.png",
		"./docs/manual.pdf":   "https://s3.amazonaws.com/bucket/docs/manual.pdf",
		"./images/banner.jpg": "https://s3.amazonaws.com/bucket/images/banner.jpg",
		"./docs/guide.docx":   "https://s3.amazonaws.com/bucket/docs/guide.docx",
	}

	// Create URL replacer
	replacer := NewURLReplacer(false, false, false)

	// Update references
	err = replacer.UpdateMediaReferences(refs, urlMapping)
	if err != nil {
		t.Fatalf("UpdateMediaReferences failed: %v", err)
	}

	// Read updated content
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	content := string(updatedContent)

	// Verify replacements
	expectedReplacements := map[string]string{
		"./images/logo.png":   "https://s3.amazonaws.com/bucket/images/logo.png",
		"./docs/manual.pdf":   "https://s3.amazonaws.com/bucket/docs/manual.pdf",
		"./images/banner.jpg": "https://s3.amazonaws.com/bucket/images/banner.jpg",
		"./docs/guide.docx":   "https://s3.amazonaws.com/bucket/docs/guide.docx",
	}

	for oldURL, newURL := range expectedReplacements {
		if strings.Contains(content, oldURL) {
			t.Errorf("Old URL %s still found in content", oldURL)
		}
		if !strings.Contains(content, newURL) {
			t.Errorf("New URL %s not found in content", newURL)
		}
	}

	// Verify specific pattern replacements
	testCases := []struct {
		description string
		pattern     string
		shouldExist bool
	}{
		{"Markdown image with new URL", "![Image](https://s3.amazonaws.com/bucket/images/logo.png)", true},
		{"Markdown link with new URL", "[Manual](https://s3.amazonaws.com/bucket/docs/manual.pdf)", true},
		{"HTML img with new URL", `<img src="https://s3.amazonaws.com/bucket/images/banner.jpg"`, true},
		{"HTML anchor with new URL", `<a href="https://s3.amazonaws.com/bucket/docs/guide.docx"`, true},
		{"Old markdown image URL", "![Image](./images/logo.png)", false},
		{"Old HTML img URL", `<img src="./images/banner.jpg"`, false},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			contains := strings.Contains(content, tc.pattern)
			if tc.shouldExist && !contains {
				t.Errorf("Expected pattern not found: %s", tc.pattern)
			} else if !tc.shouldExist && contains {
				t.Errorf("Unexpected pattern found: %s", tc.pattern)
			}
		})
	}
}

// Test ReplaceURLsInFile with enhanced patterns
func TestURLReplacer_ReplaceURLsInFile_EnhancedPatterns(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "url-replacer-enhanced-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testCases := []struct {
		name            string
		originalContent string
		replacements    map[string]string
		expectedContent string
	}{
		{
			name: "HTML embed and object tags",
			originalContent: `<embed src="./doc.pdf" type="application/pdf">
<object data="./presentation.pptx" type="application/vnd.ms-powerpoint">
<iframe src="./page.html"></iframe>`,
			replacements: map[string]string{
				"./doc.pdf":           "https://s3.example.com/doc.pdf",
				"./presentation.pptx": "https://s3.example.com/presentation.pptx",
				"./page.html":         "https://s3.example.com/page.html",
			},
			expectedContent: `<embed src="https://s3.example.com/doc.pdf" type="application/pdf">
<object data="https://s3.example.com/presentation.pptx" type="application/vnd.ms-powerpoint">
<iframe src="https://s3.example.com/page.html"></iframe>`,
		},
		{
			name: "YAML document fields",
			originalContent: `---
document: "./manual.pdf"
download_link: './guide.docx'
attachment: "file.zip"
---`,
			replacements: map[string]string{
				"./manual.pdf": "https://s3.example.com/manual.pdf",
				"./guide.docx": "https://s3.example.com/guide.docx",
				"file.zip":     "https://s3.example.com/file.zip",
			},
			expectedContent: `---
document: "https://s3.example.com/manual.pdf"
download_link: 'https://s3.example.com/guide.docx'
attachment: "https://s3.example.com/file.zip"
---`,
		},
		{
			name: "Hugo document shortcodes",
			originalContent: `{{< download src="./file.pdf" >}}
{{< document file="./doc.docx" >}}
{{< attachment url="./data.xlsx" >}}`,
			replacements: map[string]string{
				"./file.pdf":  "https://s3.example.com/file.pdf",
				"./doc.docx":  "https://s3.example.com/doc.docx",
				"./data.xlsx": "https://s3.example.com/data.xlsx",
			},
			expectedContent: `{{< download src="https://s3.example.com/file.pdf" >}}
{{< document file="https://s3.example.com/doc.docx" >}}
{{< attachment url="https://s3.example.com/data.xlsx" >}}`,
		},
		{
			name: "CSS background shorthand",
			originalContent: `.hero {
    background: url('./bg.jpg') no-repeat center;
}
.banner {
    background-image: url("./banner.png");
}`,
			replacements: map[string]string{
				"./bg.jpg":     "https://s3.example.com/bg.jpg",
				"./banner.png": "https://s3.example.com/banner.png",
			},
			expectedContent: `.hero {
    background: url('https://s3.example.com/bg.jpg') no-repeat center;
}
.banner {
    background-image: url("https://s3.example.com/banner.png");
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tc.name+".test")

			// Write original content
			err = os.WriteFile(testFile, []byte(tc.originalContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create replacer and update file
			replacer := NewURLReplacer(false, false, false)
			err = replacer.ReplaceURLsInFile(testFile, tc.replacements)
			if err != nil {
				t.Fatalf("ReplaceURLsInFile failed: %v", err)
			}

			// Read updated content
			updatedBytes, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read updated file: %v", err)
			}

			updatedContent := string(updatedBytes)
			if updatedContent != tc.expectedContent {
				t.Errorf("Content mismatch.\nExpected:\n%s\nGot:\n%s", tc.expectedContent, updatedContent)
			}
		})
	}
}

// Test ValidateMediaFileAccess
func TestURLReplacer_ValidateMediaFileAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "validate-access-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	validFile := filepath.Join(tempDir, "valid.md")
	err = os.WriteFile(validFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	readOnlyFile := filepath.Join(tempDir, "readonly.md")
	err = os.WriteFile(readOnlyFile, []byte("test content"), 0444) // Read-only
	if err != nil {
		t.Fatalf("Failed to create read-only test file: %v", err)
	}

	// Create media references
	refs := []types.MediaReference{
		{
			SourceFile:  validFile,
			LocalPath:   "./image.jpg",
			OriginalURL: "./image.jpg",
			MediaType:   types.MediaTypeImage,
		},
		{
			SourceFile:  filepath.Join(tempDir, "nonexistent.md"),
			LocalPath:   "./doc.pdf",
			OriginalURL: "./doc.pdf",
			MediaType:   types.MediaTypeDocument,
		},
	}

	replacer := NewURLReplacer(false, false, false)

	// Test with valid file
	validRefs := refs[:1]
	err = replacer.ValidateMediaFileAccess(validRefs)
	if err != nil {
		t.Errorf("Expected no error for valid file, got: %v", err)
	}

	// Test with non-existent file
	invalidRefs := refs[1:]
	err = replacer.ValidateMediaFileAccess(invalidRefs)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test dry run mode (should not check write permissions)
	dryRunReplacer := NewURLReplacer(false, true, false)
	readOnlyRefs := []types.MediaReference{
		{
			SourceFile:  readOnlyFile,
			LocalPath:   "./image.jpg",
			OriginalURL: "./image.jpg",
			MediaType:   types.MediaTypeImage,
		},
	}

	err = dryRunReplacer.ValidateMediaFileAccess(readOnlyRefs)
	if err != nil {
		t.Errorf("Dry run should not check write permissions, got: %v", err)
	}
}

// Test backward compatibility with UpdateImageReferences
func TestURLReplacer_UpdateImageReferences_BackwardCompatibility(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backward-compat-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.md")
	originalContent := `![Logo](./logo.png)
![Banner](./banner.jpg)`

	err = os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create image references (legacy format)
	refs := []types.ImageReference{
		{
			SourceFile:  testFile,
			LocalPath:   "./logo.png",
			OriginalURL: "./logo.png",
		},
		{
			SourceFile:  testFile,
			LocalPath:   "./banner.jpg",
			OriginalURL: "./banner.jpg",
		},
	}

	// Create URL mapping
	urlMapping := map[string]string{
		"./logo.png":   "https://s3.example.com/logo.png",
		"./banner.jpg": "https://s3.example.com/banner.jpg",
	}

	// Test legacy method
	replacer := NewURLReplacer(false, false, false)
	err = replacer.UpdateImageReferences(refs, urlMapping)
	if err != nil {
		t.Fatalf("UpdateImageReferences failed: %v", err)
	}

	// Verify updates
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	content := string(updatedContent)
	if !strings.Contains(content, "https://s3.example.com/logo.png") {
		t.Error("Logo URL not updated")
	}
	if !strings.Contains(content, "https://s3.example.com/banner.jpg") {
		t.Error("Banner URL not updated")
	}
}
