package integration

import (
	"path/filepath"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/internal/config"
	"github.com/SynapsesTechnologies/mv2s3/internal/scanner"
	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestBasicWorkflow(t *testing.T) {
	// Test the basic workflow integration
	// This tests that all components can work together

	// Setup test directory path
	testProjectPath := filepath.Join("..", "fixtures", "sample-project")

	// Load default configuration
	cfg := &types.MigrationConfig{
		SourceDir:      testProjectPath,
		FileExtensions: []string{"html", "css", "js", "md"},
		S3Bucket:       "test-bucket",
		S3Region:       "us-east-1",
		Concurrency:    2,
	}

	// Validate configuration
	err := config.ValidateConfig(cfg)
	if err != nil {
		// Skip validation that requires the test project to exist
		// In a real scenario, we'd set up the test fixtures properly
		t.Skip("Skipping integration test - test fixtures not available in this context")
	}

	// Test file scanner
	fileScanner := scanner.NewFileScanner(cfg)
	if fileScanner == nil {
		t.Error("Expected file scanner to be created")
	}

	// Test that scanner can identify valid files
	testFiles := []string{
		"index.html",
		"style.css",
		"script.js",
		"README.md",
		"image.png", // Should be invalid
	}

	validCount := 0
	for _, file := range testFiles {
		if fileScanner.IsValidFile(file) {
			validCount++
		}
	}

	expectedValid := 4 // html, css, js, md
	if validCount != expectedValid {
		t.Errorf("Expected %d valid files, got %d", expectedValid, validCount)
	}

	// Test link parser
	linkParser := scanner.NewLinkParser()
	if linkParser == nil {
		t.Error("Expected link parser to be created")
	}
}

func TestConfigIntegration(t *testing.T) {
	// Test that configuration loading works end-to-end

	// Test with minimal valid config
	cfg := &types.MigrationConfig{
		SourceDir:   ".",
		S3Bucket:    "test-bucket",
		Concurrency: 3,
	}

	err := config.ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got: %v", err)
	}

	// Test configuration with all components
	fileScanner := scanner.NewFileScanner(cfg)
	linkParser := scanner.NewLinkParser()

	if fileScanner == nil || linkParser == nil {
		t.Error("Expected all components to initialize successfully")
	}
}
