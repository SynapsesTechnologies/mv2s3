package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLinkParser(t *testing.T) {
	parser := NewLinkParser()
	if parser == nil {
		t.Error("Expected NewLinkParser to return a non-nil parser")
	}
}

func TestIsLocalURL(t *testing.T) {
	parser := NewLinkParser()

	testCases := []struct {
		url      string
		expected bool
	}{
		{"images/logo.png", true},
		{"./images/logo.png", true},
		{"../images/logo.png", true},
		{"https://example.com/image.jpg", false},
		{"http://example.com/image.jpg", false},
		{"//example.com/image.jpg", false},
		{"data:image/png;base64,abc", false},
		{"mailto:test@example.com", false},
	}

	for _, tc := range testCases {
		result := parser.IsLocalURL(tc.url)
		if result != tc.expected {
			t.Errorf("IsLocalURL(%s) = %v, expected %v", tc.url, result, tc.expected)
		}
	}
}

func TestParseFile(t *testing.T) {
	parser := NewLinkParser()

	// Create a temporary test file
	tempDir, err := os.MkdirTemp("", "link-parser-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.html")
	content := `<img src="./images/logo.png" alt="Logo">`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	refs, err := parser.ParseFile(testFile)
	if err != nil {
		t.Errorf("Expected no error from ParseFile, got: %v", err)
	}

	if len(refs) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(refs))
	}

	if len(refs) > 0 && refs[0].LocalPath != "./images/logo.png" {
		t.Errorf("Expected path './images/logo.png', got '%s'", refs[0].LocalPath)
	}
}
