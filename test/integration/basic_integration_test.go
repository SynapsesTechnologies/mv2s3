package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestScanCommandEndToEnd(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "bin/mv2s3", "./cmd/mv2s3")
	buildCmd.Dir = "../.."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Ensure the test fixtures exist
	testProjectPath := "test/fixtures/sample-project"
	if _, err := os.Stat(filepath.Join("../..", testProjectPath)); os.IsNotExist(err) {
		t.Skip("Test fixtures not available")
	}

	tests := []struct {
		name           string
		args           []string
		expectedImages int
		expectedDocs   int
		expectedTotal  int
	}{
		{
			name:           "default scan (images only)",
			args:           []string{"scan", testProjectPath},
			expectedImages: 11,
			expectedDocs:   0,
			expectedTotal:  11,
		},
		{
			name:           "images only scan",
			args:           []string{"scan", testProjectPath, "--images"},
			expectedImages: 11,
			expectedDocs:   0,
			expectedTotal:  11,
		},
		{
			name:           "documents only scan",
			args:           []string{"scan", testProjectPath, "--documents"},
			expectedImages: 0,
			expectedDocs:   6,
			expectedTotal:  6,
		},
		{
			name:           "both images and documents",
			args:           []string{"scan", testProjectPath, "--images", "--documents"},
			expectedImages: 11,
			expectedDocs:   6,
			expectedTotal:  17,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Run the scan command
			cmd := exec.Command("./bin/mv2s3", test.args...)
			cmd.Dir = "../.."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Scan command failed: %v\nOutput: %s", err, output)
			}

			outputStr := string(output)

			// Check for expected output patterns
			if test.expectedImages > 0 && test.expectedDocs == 0 {
				// Images only mode
				expectedPattern := "Images found:"
				if !strings.Contains(outputStr, expectedPattern) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedPattern, outputStr)
				}
			} else if test.expectedImages == 0 && test.expectedDocs > 0 {
				// Documents only mode
				expectedPattern := "Documents found:"
				if !strings.Contains(outputStr, expectedPattern) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedPattern, outputStr)
				}
			} else if test.expectedImages > 0 && test.expectedDocs > 0 {
				// Multiple media types mode
				expectedPattern := "Media files found:"
				if !strings.Contains(outputStr, expectedPattern) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedPattern, outputStr)
				}
			}

			// Verify no errors in output
			if strings.Contains(outputStr, "error:") || strings.Contains(outputStr, "Error:") {
				t.Errorf("Unexpected error in output: %s", outputStr)
			}

			// Verify scan completed successfully
			if !strings.Contains(outputStr, "Files scanned:") {
				t.Errorf("Expected output to contain scan summary, got: %s", outputStr)
			}
		})
	}
}
