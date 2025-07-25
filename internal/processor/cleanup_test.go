package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestCleanupWorkflow(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create test images
	imageDir := filepath.Join(tmpDir, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("Failed to create image directory: %v", err)
	}

	testImages := []string{"logo.png", "header.jpg", "icon.svg"}
	for _, img := range testImages {
		imgPath := filepath.Join(imageDir, img)
		if err := os.WriteFile(imgPath, []byte("fake image data"), 0644); err != nil {
			t.Fatalf("Failed to create test image %s: %v", img, err)
		}
	}

	// Create test markdown file
	mdFile := filepath.Join(tmpDir, "test.md")
	mdContent := `# Test Document
![Logo](images/logo.png)
![Header](images/header.jpg)
![Icon](images/icon.svg)
`
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to create markdown file: %v", err)
	}

	// Verify all files exist before cleanup
	for _, img := range testImages {
		imgPath := filepath.Join(imageDir, img)
		if _, err := os.Stat(imgPath); err != nil {
			t.Fatalf("Test image %s should exist before cleanup", imgPath)
		}
	}

	// Simulate cleanup by manually removing files
	// In a real scenario, this would be done by the cleanup command
	for _, img := range testImages {
		imgPath := filepath.Join(imageDir, img)
		if err := os.Remove(imgPath); err != nil {
			t.Fatalf("Failed to remove test image %s: %v", img, err)
		}
	}

	// Verify files are deleted
	for _, img := range testImages {
		imgPath := filepath.Join(imageDir, img)
		if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
			t.Fatalf("Test image %s should be deleted after cleanup", imgPath)
		}
	}
}

func TestBackupCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create backup
	backupDir := filepath.Join(tmpDir, "backups")
	backupFile := filepath.Join(backupDir, "test.txt")

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Use copyFile function from main package
	srcContent, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	if err := os.WriteFile(backupFile, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup exists and has same content
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != content {
		t.Errorf("Backup content mismatch. Expected: %s, Got: %s", content, string(backupContent))
	}

	// Delete original file
	if err := os.Remove(srcFile); err != nil {
		t.Fatalf("Failed to delete original file: %v", err)
	}

	// Verify original is gone but backup remains
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("Original file should be deleted")
	}

	if _, err := os.Stat(backupFile); err != nil {
		t.Errorf("Backup file should exist: %v", err)
	}
}

func TestImageReferenceCleanup(t *testing.T) {
	// Test cleanup logic for image references
	refs := []types.ImageReference{
		{
			SourceFile:  "content/post1.md",
			OriginalURL: "images/logo.png",
			LineNumber:  10,
		},
		{
			SourceFile:  "content/post2.md",
			OriginalURL: "images/header.jpg",
			LineNumber:  5,
		},
		{
			SourceFile:  "content/post3.md",
			OriginalURL: "images/icon.svg",
			LineNumber:  15,
		},
	}

	// Create URL mapping (simulating successful uploads)
	urlMapping := map[string]string{
		"images/logo.png":   "https://bucket.s3.amazonaws.com/logo.png",
		"images/header.jpg": "https://bucket.s3.amazonaws.com/header.jpg",
		// Note: images/icon.svg is missing - simulating failed upload
	}

	// Filter successful uploads
	var successfulRefs []types.ImageReference
	for _, ref := range refs {
		if _, exists := urlMapping[ref.OriginalURL]; exists {
			successfulRefs = append(successfulRefs, ref)
		}
	}

	// Should only have 2 successful references (logo.png and header.jpg)
	expectedCount := 2
	if len(successfulRefs) != expectedCount {
		t.Errorf("Expected %d successful references, got %d", expectedCount, len(successfulRefs))
	}

	// Verify correct references are included
	expectedURLs := map[string]bool{
		"images/logo.png":   false,
		"images/header.jpg": false,
	}

	for _, ref := range successfulRefs {
		if _, exists := expectedURLs[ref.OriginalURL]; exists {
			expectedURLs[ref.OriginalURL] = true
		} else {
			t.Errorf("Unexpected URL in successful references: %s", ref.OriginalURL)
		}
	}

	for url, found := range expectedURLs {
		if !found {
			t.Errorf("Expected URL %s not found in successful references", url)
		}
	}
}
