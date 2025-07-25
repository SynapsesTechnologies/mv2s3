package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CreateTempProject creates a temporary test project structure
func CreateTempProject(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create directories
	dirs := []string{
		"images",
		"css",
		"js",
		"images/icons",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	// Create test files
	files := map[string]string{
		"index.html": `<!DOCTYPE html>
<html>
<head>
    <title>Test</title>
    <link rel="icon" href="images/favicon.ico">
</head>
<body>
    <img src="images/logo.png" alt="Logo">
    <img src="./images/banner.jpg" alt="Banner">
</body>
</html>`,
		"css/style.css": `.logo {
    background-image: url('../images/logo.png');
}
.hero {
    background: url("../images/hero-bg.jpg");
}`,
		"js/main.js": `const logo = './images/logo.png';
const banner = 'images/banner.jpg';`,
		"README.md": `# Test Project

![Logo](images/logo.png)
![Banner](./images/banner.jpg)`,
	}

	for filePath, content := range files {
		fullPath := filepath.Join(tmpDir, filePath)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}
	}

	// Create dummy image files
	imageFiles := []string{
		"images/favicon.ico",
		"images/logo.png",
		"images/banner.jpg",
		"images/hero-bg.jpg",
		"images/icons/home.svg",
	}

	for _, imgFile := range imageFiles {
		fullPath := filepath.Join(tmpDir, imgFile)
		err := os.WriteFile(fullPath, []byte("dummy image content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test image %s: %v", imgFile, err)
		}
	}

	return tmpDir
}

// AssertFileExists checks if a file exists and fails the test if it doesn't
func AssertFileExists(t *testing.T, filePath string) {
	t.Helper()

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", filePath)
	}
}

// AssertFileNotExists checks if a file doesn't exist and fails the test if it does
func AssertFileNotExists(t *testing.T, filePath string) {
	t.Helper()

	if _, err := os.Stat(filePath); err == nil {
		t.Errorf("Expected file to not exist: %s", filePath)
	}
}

// AssertFileContains checks if a file contains specific content
func AssertFileContains(t *testing.T, filePath, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath, err)
	}

	if !strings.Contains(string(content), expectedContent) {
		t.Errorf("File %s does not contain expected content: %s", filePath, expectedContent)
	}
}
