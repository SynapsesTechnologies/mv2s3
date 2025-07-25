package types

import (
	"testing"
	"time"
)

func TestImageReference(t *testing.T) {
	testTime := time.Now()
	ref := ImageReference{
		SourceFile:   "index.html",
		LocalPath:    "images/logo.png",
		LineNumber:   10,
		OriginalURL:  "./images/logo.png",
		NewURL:       "https://bucket.s3.amazonaws.com/images/logo.png",
		FileSize:     1024,
		ContentType:  "image/png",
		LastModified: testTime,
	}

	if ref.SourceFile != "index.html" {
		t.Errorf("Expected SourceFile to be 'index.html', got %s", ref.SourceFile)
	}

	if ref.LocalPath != "images/logo.png" {
		t.Errorf("Expected LocalPath to be 'images/logo.png', got %s", ref.LocalPath)
	}

	if ref.LineNumber != 10 {
		t.Errorf("Expected LineNumber to be 10, got %d", ref.LineNumber)
	}

	if ref.OriginalURL != "./images/logo.png" {
		t.Errorf("Expected OriginalURL to be './images/logo.png', got %s", ref.OriginalURL)
	}

	if ref.NewURL != "https://bucket.s3.amazonaws.com/images/logo.png" {
		t.Errorf("Expected NewURL to be 'https://bucket.s3.amazonaws.com/images/logo.png', got %s", ref.NewURL)
	}

	if ref.FileSize != 1024 {
		t.Errorf("Expected FileSize to be 1024, got %d", ref.FileSize)
	}

	if ref.ContentType != "image/png" {
		t.Errorf("Expected ContentType to be 'image/png', got %s", ref.ContentType)
	}

	if !ref.LastModified.Equal(testTime) {
		t.Errorf("Expected LastModified to be %v, got %v", testTime, ref.LastModified)
	}
}

func TestMigrationConfig_DefaultValues(t *testing.T) {
	config := MigrationConfig{
		SourceDir:      ".",
		FileExtensions: []string{"html", "css", "js"},
		S3Region:       "us-east-1",
		Concurrency:    5,
		LogLevel:       "info",
	}

	if config.SourceDir != "." {
		t.Errorf("Expected SourceDir to be '.', got %s", config.SourceDir)
	}

	if len(config.FileExtensions) != 3 {
		t.Errorf("Expected 3 file extensions, got %d", len(config.FileExtensions))
	}

	if config.S3Region != "us-east-1" {
		t.Errorf("Expected S3Region to be 'us-east-1', got %s", config.S3Region)
	}

	if config.Concurrency != 5 {
		t.Errorf("Expected Concurrency to be 5, got %d", config.Concurrency)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got %s", config.LogLevel)
	}
}

func TestMigrationError(t *testing.T) {
	// Test basic error
	err := &MigrationError{
		Type:    ErrorTypeValidation,
		Message: "invalid configuration",
	}

	expected := "invalid configuration"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test error with file
	err = &MigrationError{
		Type:    ErrorTypeParsing,
		Message: "parse error",
		File:    "index.html",
	}

	expected = "index.html: parse error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test error with file and line
	err = &MigrationError{
		Type:    ErrorTypeParsing,
		Message: "syntax error",
		File:    "main.css",
		Line:    25,
	}

	expected = "main.css:25: syntax error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestErrorTypeConstants(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		name      string
	}{
		{ErrorTypeFileSystem, "ErrorTypeFileSystem"},
		{ErrorTypeS3, "ErrorTypeS3"},
		{ErrorTypeParsing, "ErrorTypeParsing"},
		{ErrorTypeValidation, "ErrorTypeValidation"},
		{ErrorTypeNetwork, "ErrorTypeNetwork"},
	}

	for i, test := range tests {
		if int(test.errorType) != i {
			t.Errorf("Expected %s to have value %d, got %d", test.name, i, int(test.errorType))
		}
	}
}
