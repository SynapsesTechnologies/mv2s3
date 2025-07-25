package scanner

import (
	"testing"
	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestNewFileScanner(t *testing.T) {
	config := &types.MigrationConfig{
		SourceDir:      ".",
		FileExtensions: []string{"html", "css", "js"},
	}

	scanner := NewFileScanner(config)
	if scanner == nil {
		t.Error("Expected NewFileScanner to return a non-nil scanner")
	}

	if scanner.config != config {
		t.Error("Expected scanner config to match provided config")
	}
}

func TestIsValidFile(t *testing.T) {
	config := &types.MigrationConfig{
		FileExtensions: []string{"html", "css", "js", "md"},
	}

	scanner := NewFileScanner(config)

	tests := []struct {
		filePath string
		expected bool
	}{
		{"index.html", true},
		{"style.css", true},
		{"script.js", true},
		{"README.md", true},
		{"component.jsx", false}, // Not in extensions
		{"image.png", false},     // Not in extensions
		{"file.txt", false},      // Not in extensions
		{"noextension", false},   // No extension
		{"index.HTML", true},     // Case insensitive
		{"STYLE.CSS", true},      // Case insensitive
	}

	for _, test := range tests {
		result := scanner.IsValidFile(test.filePath)
		if result != test.expected {
			t.Errorf("IsValidFile(%s) = %v, expected %v", test.filePath, result, test.expected)
		}
	}
}

func TestIsValidFile_EmptyExtensions(t *testing.T) {
	config := &types.MigrationConfig{
		FileExtensions: []string{},
	}

	scanner := NewFileScanner(config)

	// Should return false for any file when no extensions are configured
	if scanner.IsValidFile("index.html") {
		t.Error("Expected IsValidFile to return false when no extensions are configured")
	}
}
