package types

import (
	"fmt"
	"time"
)

// ImageReference represents a found image reference in source code
type ImageReference struct {
	SourceFile   string    // Path to the source file containing the reference
	LocalPath    string    // Original local path to the image
	LineNumber   int       // Line number where reference was found
	OriginalURL  string    // Original URL/path as found in source
	NewURL       string    // New S3 URL after upload
	FileSize     int64     // Size of the image file in bytes
	ContentType  string    // MIME type of the image
	LastModified time.Time // Last modified time of local file
}

// MigrationConfig holds all configuration for the migration process
type MigrationConfig struct {
	// Source scanning
	SourceDir       string   `mapstructure:"source_dir" yaml:"source_dir"`
	FileExtensions  []string `mapstructure:"file_extensions" yaml:"file_extensions"`
	ExcludePatterns []string `mapstructure:"exclude_patterns" yaml:"exclude_patterns"`
	IncludePatterns []string `mapstructure:"include_patterns" yaml:"include_patterns"`

	// AWS S3 configuration
	S3Bucket    string `mapstructure:"s3_bucket" yaml:"s3_bucket"`
	S3Region    string `mapstructure:"s3_region" yaml:"s3_region"`
	S3Prefix    string `mapstructure:"s3_prefix" yaml:"s3_prefix"`
	S3Endpoint  string `mapstructure:"s3_endpoint" yaml:"s3_endpoint"` // For S3-compatible services
	S3AccessKey string `mapstructure:"s3_access_key" yaml:"s3_access_key"`
	S3SecretKey string `mapstructure:"s3_secret_key" yaml:"s3_secret_key"`
	S3Profile   string `mapstructure:"s3_profile" yaml:"s3_profile"`

	// Processing options
	DryRun          bool   `mapstructure:"dry_run" yaml:"dry_run"`
	CleanupLocal    bool   `mapstructure:"cleanup_local" yaml:"cleanup_local"`
	BackupOriginals bool   `mapstructure:"backup_originals" yaml:"backup_originals"`
	BackupDir       string `mapstructure:"backup_dir" yaml:"backup_dir"`
	Concurrency     int    `mapstructure:"concurrency" yaml:"concurrency"`

	// Logging
	LogLevel  string `mapstructure:"log_level" yaml:"log_level"`
	LogFormat string `mapstructure:"log_format" yaml:"log_format"`
	Verbose   bool   `mapstructure:"verbose" yaml:"verbose"`
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	TotalFiles     int
	ProcessedFiles int
	UploadedImages int
	UpdatedFiles   int
	Errors         []error
	Duration       time.Duration
}

// ErrorType represents different categories of errors
type ErrorType int

const (
	ErrorTypeFileSystem ErrorType = iota
	ErrorTypeS3
	ErrorTypeParsing
	ErrorTypeValidation
	ErrorTypeNetwork
)

// MigrationError represents a specific error during migration
type MigrationError struct {
	Type     ErrorType
	Message  string
	File     string
	Line     int
	Original error
}

func (e *MigrationError) Error() string {
	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
	} else if e.File != "" {
		return fmt.Sprintf("%s: %s", e.File, e.Message)
	}
	return e.Message
}

func (e *MigrationError) Unwrap() error {
	return e.Original
}
