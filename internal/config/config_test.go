package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Test loading config with defaults (no config file)
	config, _, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Expected no error loading default config, got: %v", err)
	}

	// Check default values
	if config.SourceDir != "." {
		t.Errorf("Expected default SourceDir to be '.', got %s", config.SourceDir)
	}

	if config.S3Region != "us-east-1" {
		t.Errorf("Expected default S3Region to be 'us-east-1', got %s", config.S3Region)
	}

	if config.Concurrency != 5 {
		t.Errorf("Expected default Concurrency to be 5, got %d", config.Concurrency)
	}

	if !config.BackupOriginals {
		t.Error("Expected BackupOriginals to be true by default")
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be 'info', got %s", config.LogLevel)
	}
}

func TestLoadConfig_WithConfigFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
source_dir: "/test/project"
s3_bucket: "test-bucket"
s3_region: "eu-west-1"
s3_prefix: "assets/"
concurrency: 10
dry_run: true
cleanup_local: true
log_level: "debug"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, _, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Expected no error loading config file, got: %v", err)
	}

	// Check loaded values
	if config.SourceDir != "/test/project" {
		t.Errorf("Expected SourceDir to be '/test/project', got %s", config.SourceDir)
	}

	if config.S3Bucket != "test-bucket" {
		t.Errorf("Expected S3Bucket to be 'test-bucket', got %s", config.S3Bucket)
	}

	if config.S3Region != "eu-west-1" {
		t.Errorf("Expected S3Region to be 'eu-west-1', got %s", config.S3Region)
	}

	if config.Concurrency != 10 {
		t.Errorf("Expected Concurrency to be 10, got %d", config.Concurrency)
	}

	if !config.DryRun {
		t.Error("Expected DryRun to be true")
	}

	if !config.CleanupLocal {
		t.Error("Expected CleanupLocal to be true")
	}
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	config := &types.MigrationConfig{
		SourceDir:       tmpDir,
		S3Bucket:        "test-bucket",
		Concurrency:     5,
		BackupOriginals: false, // Disable backup for this test
	}

	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestValidateConfig_MissingSourceDir(t *testing.T) {
	config := &types.MigrationConfig{
		SourceDir:   "/nonexistent/directory",
		S3Bucket:    "test-bucket",
		Concurrency: 5,
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for nonexistent source directory")
	}

	migErr, ok := err.(*types.MigrationError)
	if !ok {
		t.Errorf("Expected MigrationError, got %T", err)
	}

	if migErr.Type != types.ErrorTypeValidation {
		t.Errorf("Expected ErrorTypeValidation, got %v", migErr.Type)
	}
}

func TestValidateConfig_MissingBucket(t *testing.T) {
	tmpDir := t.TempDir()

	config := &types.MigrationConfig{
		SourceDir:   tmpDir,
		S3Bucket:    "", // Missing bucket
		Concurrency: 5,
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for missing S3 bucket")
	}

	migErr, ok := err.(*types.MigrationError)
	if !ok {
		t.Errorf("Expected MigrationError, got %T", err)
	}

	if migErr.Type != types.ErrorTypeValidation {
		t.Errorf("Expected ErrorTypeValidation, got %v", migErr.Type)
	}
}

func TestValidateConfig_InvalidConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []int{0, -1, 25, 100}

	for _, concurrency := range tests {
		config := &types.MigrationConfig{
			SourceDir:   tmpDir,
			S3Bucket:    "test-bucket",
			Concurrency: concurrency,
		}

		err := ValidateConfig(config)
		if err == nil {
			t.Errorf("Expected error for invalid concurrency %d", concurrency)
		}

		migErr, ok := err.(*types.MigrationError)
		if !ok {
			t.Errorf("Expected MigrationError, got %T", err)
		}

		if migErr.Type != types.ErrorTypeValidation {
			t.Errorf("Expected ErrorTypeValidation, got %v", migErr.Type)
		}
	}
}

func TestValidateConfig_BackupDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")

	config := &types.MigrationConfig{
		SourceDir:       tmpDir,
		S3Bucket:        "test-bucket",
		Concurrency:     5,
		BackupOriginals: true,
		BackupDir:       backupDir,
	}

	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error when creating backup directory, got: %v", err)
	}

	// Check that backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Expected backup directory to be created")
	}
}
