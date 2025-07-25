package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// TestEndToEndIntegration tests the complete scanning workflow
func TestEndToEndIntegration(t *testing.T) {
	// Create temporary project structure
	tempDir, err := os.MkdirTemp("", "mv2s3-e2e-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create project structure
	dirs := []string{
		"src",
		"docs",
		"assets/images",
		"assets/downloads",
	}
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(tempDir, dir), 0755)
	}

	// Create source files with media references
	sourceFiles := map[string]string{
		"README.md": `# Project Documentation

![Logo](./assets/images/logo.png)
![Banner](./assets/images/banner.jpg)

## Downloads
- [User Manual](./docs/manual.pdf)
- [Quick Guide](./docs/quick-guide.docx)
- [Data Template](./assets/downloads/template.xlsx)

## Media
See our [promotional video](./assets/videos/promo.mp4).
`,
		"docs/guide.md": `# Guide

![Diagram](../assets/images/architecture.svg)
[Download PDF](./technical-spec.pdf)
`,
		"src/index.html": `<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="../assets/styles.css">
</head>
<body>
    <img src="../assets/images/hero.jpg" alt="Hero">
    <a href="../docs/whitepaper.pdf">Download Whitepaper</a>
    <embed src="../assets/presentations/demo.pptx" type="application/vnd.ms-powerpoint">
</body>
</html>`,
		"assets/styles.css": `body {
    background-image: url('./images/background.png');
}

.hero {
    background: url('./images/hero-bg.jpg') no-repeat;
}`,
	}

	for filePath, content := range sourceFiles {
		fullPath := filepath.Join(tempDir, filePath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file %s: %v", filePath, err)
		}
	}

	// Create actual media files
	mediaFiles := []string{
		"assets/images/logo.png",
		"assets/images/banner.jpg",
		"assets/images/architecture.svg",
		"assets/images/hero.jpg",
		"assets/images/background.png",
		"assets/images/hero-bg.jpg",
		"docs/manual.pdf",
		"docs/quick-guide.docx",
		"docs/technical-spec.pdf",
		"docs/whitepaper.pdf",
		"assets/downloads/template.xlsx",
		"assets/presentations/demo.pptx",
		"assets/videos/promo.mp4",
	}

	for _, mediaFile := range mediaFiles {
		fullPath := filepath.Join(tempDir, mediaFile)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		err = os.WriteFile(fullPath, []byte("fake content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create media file %s: %v", mediaFile, err)
		}
	}

	// Test 1: Scan with only images enabled (legacy behavior)
	t.Run("ImagesOnly", func(t *testing.T) {
		config := &types.MigrationConfig{
			SourceDir:      tempDir,
			FileExtensions: []string{"md", "html", "css"},
			MediaTypes: map[types.MediaType]*types.MediaConfig{
				types.MediaTypeImage: {Enabled: true},
			},
		}

		scanner := NewFileScanner(config)
		references, err := scanner.ScanForMediaReferences()
		if err != nil {
			t.Fatalf("Failed to scan media references: %v", err)
		}

		// Should find all image references
		expectedImages := 6 // logo, banner, architecture, hero, background, hero-bg
		actualImages := 0
		for _, ref := range references {
			if ref.MediaType == types.MediaTypeImage {
				actualImages++
			}
		}

		if actualImages != expectedImages {
			t.Errorf("Expected %d image references, got %d", expectedImages, actualImages)
		}

		// Should not find any document references
		for _, ref := range references {
			if ref.MediaType == types.MediaTypeDocument {
				t.Errorf("Found unexpected document reference: %s", ref.LocalPath)
			}
		}
	})

	// Test 2: Scan with all media types enabled
	t.Run("AllMediaTypes", func(t *testing.T) {
		config := &types.MigrationConfig{
			SourceDir:      tempDir,
			FileExtensions: []string{"md", "html", "css"},
			MediaTypes: map[types.MediaType]*types.MediaConfig{
				types.MediaTypeImage:    {Enabled: true},
				types.MediaTypeDocument: {Enabled: true},
				types.MediaTypeVideo:    {Enabled: true},
			},
		}

		scanner := NewFileScanner(config)
		references, err := scanner.ScanForMediaReferences()
		if err != nil {
			t.Fatalf("Failed to scan media references: %v", err)
		}

		// Count references by type
		counts := make(map[types.MediaType]int)
		for _, ref := range references {
			counts[ref.MediaType]++
		}

		// Expected counts
		expectedCounts := map[types.MediaType]int{
			types.MediaTypeImage:    6, // logo, banner, architecture, hero, background, hero-bg
			types.MediaTypeDocument: 6, // manual, quick-guide, technical-spec, whitepaper, template, demo
			types.MediaTypeVideo:    1, // promo
		}

		for mediaType, expectedCount := range expectedCounts {
			if counts[mediaType] != expectedCount {
				t.Errorf("Expected %d %s references, got %d",
					expectedCount, mediaType, counts[mediaType])
			}
		}
	})

	// Test 3: Scan actual media files
	t.Run("ScanMediaFiles", func(t *testing.T) {
		config := &types.MigrationConfig{
			SourceDir: tempDir,
			MediaTypes: map[types.MediaType]*types.MediaConfig{
				types.MediaTypeImage:    {Enabled: true},
				types.MediaTypeDocument: {Enabled: true},
				types.MediaTypeVideo:    {Enabled: true},
			},
		}

		scanner := NewFileScanner(config)
		mediaFiles, err := scanner.ScanMediaFiles()
		if err != nil {
			t.Fatalf("Failed to scan media files: %v", err)
		}

		// Expected file counts
		expectedCounts := map[types.MediaType]int{
			types.MediaTypeImage:    6,
			types.MediaTypeDocument: 6,
			types.MediaTypeVideo:    1,
		}

		for mediaType, expectedCount := range expectedCounts {
			if len(mediaFiles[mediaType]) != expectedCount {
				t.Errorf("Expected %d %s files, got %d",
					expectedCount, mediaType, len(mediaFiles[mediaType]))
			}
		}
	})

	// Test 4: Test backward compatibility
	t.Run("BackwardCompatibility", func(t *testing.T) {
		config := &types.MigrationConfig{
			SourceDir:      tempDir,
			FileExtensions: []string{"md", "html", "css"},
		}

		scanner := NewFileScanner(config)
		imageRefs, err := scanner.ScanForImageReferences()
		if err != nil {
			t.Fatalf("Failed to scan image references: %v", err)
		}

		// Should find all image references using legacy method
		if len(imageRefs) != 6 {
			t.Errorf("Expected 6 image references using legacy method, got %d", len(imageRefs))
		}
	})
}
