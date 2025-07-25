package types

import (
	"fmt"
	"strings"
	"time"
)

// MediaType represents different categories of media files
type MediaType int

const (
	MediaTypeImage MediaType = iota
	MediaTypeDocument
	MediaTypeVideo
	MediaTypeAudio
	MediaTypeOther
)

// String returns the string representation of MediaType
func (mt MediaType) String() string {
	switch mt {
	case MediaTypeImage:
		return "images"
	case MediaTypeDocument:
		return "documents"
	case MediaTypeVideo:
		return "videos"
	case MediaTypeAudio:
		return "audio"
	case MediaTypeOther:
		return "other"
	default:
		return "unknown"
	}
}

// MarshalYAML implements yaml.Marshaler interface
func (mt MediaType) MarshalYAML() (interface{}, error) {
	return mt.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (mt *MediaType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	parsed, err := ParseMediaType(s)
	if err != nil {
		return err
	}

	*mt = parsed
	return nil
}

// ParseMediaType converts a string to MediaType
func ParseMediaType(s string) (MediaType, error) {
	switch strings.ToLower(s) {
	case "images", "image":
		return MediaTypeImage, nil
	case "documents", "document", "docs", "doc":
		return MediaTypeDocument, nil
	case "videos", "video":
		return MediaTypeVideo, nil
	case "audio":
		return MediaTypeAudio, nil
	case "other":
		return MediaTypeOther, nil
	default:
		return MediaTypeOther, fmt.Errorf("unknown media type: %s", s)
	}
}

// MediaConfig holds configuration for a specific media type
type MediaConfig struct {
	Enabled    bool     `mapstructure:"enabled" yaml:"enabled"`
	Extensions []string `mapstructure:"extensions" yaml:"extensions"`
	S3Bucket   string   `mapstructure:"s3_bucket" yaml:"s3_bucket"`
	S3Prefix   string   `mapstructure:"s3_prefix" yaml:"s3_prefix"`
}

// GetDefaultMediaConfig returns default configuration for a media type
func GetDefaultMediaConfig(mediaType MediaType) *MediaConfig {
	switch mediaType {
	case MediaTypeImage:
		return &MediaConfig{
			Enabled:    true,
			Extensions: []string{"jpg", "jpeg", "png", "gif", "svg", "webp", "ico", "bmp"},
			S3Bucket:   "",
			S3Prefix:   "images/",
		}
	case MediaTypeDocument:
		return &MediaConfig{
			Enabled:    false,
			Extensions: []string{"pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx", "txt", "rtf"},
			S3Bucket:   "",
			S3Prefix:   "documents/",
		}
	case MediaTypeVideo:
		return &MediaConfig{
			Enabled:    false,
			Extensions: []string{"mp4", "avi", "mov", "wmv", "flv", "webm", "mkv"},
			S3Bucket:   "",
			S3Prefix:   "videos/",
		}
	case MediaTypeAudio:
		return &MediaConfig{
			Enabled:    false,
			Extensions: []string{"mp3", "wav", "ogg", "flac", "aac", "m4a"},
			S3Bucket:   "",
			S3Prefix:   "audio/",
		}
	default:
		return &MediaConfig{
			Enabled:    false,
			Extensions: []string{},
			S3Bucket:   "",
			S3Prefix:   "other/",
		}
	}
}

// MediaReference represents a found media reference in source code (replaces ImageReference)
type MediaReference struct {
	SourceFile   string    // Path to the source file containing the reference
	LocalPath    string    // Original local path to the media file
	LineNumber   int       // Line number where reference was found
	OriginalURL  string    // Original URL/path as found in source
	NewURL       string    // New S3 URL after upload
	FileSize     int64     // Size of the media file in bytes
	ContentType  string    // MIME type of the media file
	LastModified time.Time // Last modified time of local file
	MediaType    MediaType // Type of media (image, document, etc.)
}

// ImageReference represents a found image reference in source code
// Deprecated: Use MediaReference instead. Kept for backward compatibility.
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

	// Media type configuration (v0.2.0+)
	EnabledMediaTypes    []MediaType                `yaml:"enabled_media_types,omitempty"` // Handled specially
	EnabledMediaTypesStr []string                   `mapstructure:"enabled_media_types"`   // For viper unmarshaling
	MediaTypes           map[MediaType]*MediaConfig `mapstructure:"media_types" yaml:"media_types"`

	// AWS S3 configuration (legacy, still supported for backward compatibility)
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

// ParseEnabledMediaTypes converts string representations to MediaType enums
func (mc *MigrationConfig) ParseEnabledMediaTypes() error {
	if len(mc.EnabledMediaTypesStr) == 0 {
		return nil
	}

	mc.EnabledMediaTypes = make([]MediaType, 0, len(mc.EnabledMediaTypesStr))
	for _, typeStr := range mc.EnabledMediaTypesStr {
		mediaType, err := ParseMediaType(typeStr)
		if err != nil {
			return fmt.Errorf("invalid media type '%s': %w", typeStr, err)
		}
		mc.EnabledMediaTypes = append(mc.EnabledMediaTypes, mediaType)
	}

	return nil
}

// GetMediaConfig returns the media configuration for a given media type
// Falls back to legacy S3 configuration if media-specific config is not available
func (mc *MigrationConfig) GetMediaConfig(mediaType MediaType) *MediaConfig {
	// Check if we have media-specific configuration
	if mc.MediaTypes != nil {
		if config, exists := mc.MediaTypes[mediaType]; exists {
			return config
		}
	}

	// Fall back to legacy configuration for images
	if mediaType == MediaTypeImage {
		return &MediaConfig{
			Enabled:    true,
			Extensions: GetDefaultMediaConfig(MediaTypeImage).Extensions,
			S3Bucket:   mc.S3Bucket,
			S3Prefix:   mc.S3Prefix,
		}
	}

	// Return default config for other media types
	return GetDefaultMediaConfig(mediaType)
}

// IsMediaTypeEnabled checks if a media type is enabled
func (mc *MigrationConfig) IsMediaTypeEnabled(mediaType MediaType) bool {
	// If EnabledMediaTypes is specified, use it
	if len(mc.EnabledMediaTypes) > 0 {
		for _, enabled := range mc.EnabledMediaTypes {
			if enabled == mediaType {
				return true
			}
		}
		return false
	}

	// Fall back to media-specific configuration
	config := mc.GetMediaConfig(mediaType)
	return config.Enabled
}

// GetEnabledMediaTypes returns all enabled media types
func (mc *MigrationConfig) GetEnabledMediaTypes() []MediaType {
	if len(mc.EnabledMediaTypes) > 0 {
		return mc.EnabledMediaTypes
	}

	// Fall back to checking individual media configurations
	var enabled []MediaType
	for mediaType := MediaTypeImage; mediaType <= MediaTypeOther; mediaType++ {
		if mc.IsMediaTypeEnabled(mediaType) {
			enabled = append(enabled, mediaType)
		}
	}

	// If no media types are explicitly configured, default to images only
	if len(enabled) == 0 {
		return []MediaType{MediaTypeImage}
	}

	return enabled
}

// InitializeMediaTypes ensures MediaTypes map is properly initialized with defaults
func (mc *MigrationConfig) InitializeMediaTypes() {
	if mc.MediaTypes == nil {
		mc.MediaTypes = make(map[MediaType]*MediaConfig)
	}

	// Initialize default configurations for all media types if not present
	for mediaType := MediaTypeImage; mediaType <= MediaTypeOther; mediaType++ {
		if _, exists := mc.MediaTypes[mediaType]; !exists {
			mc.MediaTypes[mediaType] = GetDefaultMediaConfig(mediaType)
		}
	}

	// Apply legacy configuration to image media type if no media-specific config exists
	if mc.S3Bucket != "" && (mc.MediaTypes[MediaTypeImage].S3Bucket == "" || mc.MediaTypes[MediaTypeImage].S3Bucket == GetDefaultMediaConfig(MediaTypeImage).S3Bucket) {
		mc.MediaTypes[MediaTypeImage].S3Bucket = mc.S3Bucket
		mc.MediaTypes[MediaTypeImage].S3Prefix = mc.S3Prefix
		mc.MediaTypes[MediaTypeImage].Enabled = true
	}
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	TotalFiles     int
	ProcessedFiles int
	UploadedMedia  map[MediaType]int // Count per media type
	UploadedImages int               // Deprecated: Use UploadedMedia[MediaTypeImage] instead
	UpdatedFiles   int
	Errors         []error
	Duration       time.Duration
}

// NewMigrationResult creates a new MigrationResult with initialized media counts
func NewMigrationResult() *MigrationResult {
	return &MigrationResult{
		UploadedMedia: make(map[MediaType]int),
	}
}

// AddUploadedMedia increments the count for a specific media type
func (mr *MigrationResult) AddUploadedMedia(mediaType MediaType, count int) {
	if mr.UploadedMedia == nil {
		mr.UploadedMedia = make(map[MediaType]int)
	}
	mr.UploadedMedia[mediaType] += count

	// Maintain backward compatibility
	if mediaType == MediaTypeImage {
		mr.UploadedImages += count
	}
}

// GetTotalUploaded returns the total number of uploaded media files
func (mr *MigrationResult) GetTotalUploaded() int {
	total := 0
	for _, count := range mr.UploadedMedia {
		total += count
	}
	return total
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
