package config

import (
	"fmt"
	"os"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
	"github.com/spf13/viper"
)

// LoadConfig loads configuration from file, environment variables, and defaults
// Returns the config and the path of the config file that was actually used
func LoadConfig(cfgFile string) (*types.MigrationConfig, string, error) {
	v := viper.New()

	// Set configuration file
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// Search for config in home directory and current directory
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
		}
		v.AddConfigPath(".")
		v.SetConfigType("yaml")
		v.SetConfigName(".mv2s3")
	}

	// Environment variable prefix
	v.SetEnvPrefix("MV2S3")
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, "", err
		}
	}

	var config types.MigrationConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, "", err
	}

	// Parse string-based media types to enum types
	if err := config.ParseEnabledMediaTypes(); err != nil {
		return nil, "", fmt.Errorf("failed to parse enabled media types: %w", err)
	}

	// Handle configuration migration from v0.1.x to v0.2.0
	migrateV1ConfigToV2(&config)

	// Initialize media types with defaults and backward compatibility
	config.InitializeMediaTypes()

	// Get the actual config file that was used
	configFileUsed := v.ConfigFileUsed()

	return &config, configFileUsed, nil
}

func setDefaults(v *viper.Viper) {
	// Source scanning defaults
	v.SetDefault("source_dir", ".")
	v.SetDefault("file_extensions", []string{"html", "css", "js", "md", "jsx", "tsx"})
	v.SetDefault("exclude_patterns", []string{"node_modules/**", ".git/**", "dist/**", "build/**"})

	// Media type defaults (v0.2.0+)
	v.SetDefault("enabled_media_types", []string{"images"}) // Default to images only for backward compatibility

	// S3 defaults (legacy, maintained for backward compatibility)
	v.SetDefault("s3_region", "us-east-1")
	v.SetDefault("s3_prefix", "")

	// Processing defaults
	v.SetDefault("dry_run", false)
	v.SetDefault("cleanup_local", false)
	v.SetDefault("backup_originals", true)
	v.SetDefault("backup_dir", "./backups")
	v.SetDefault("concurrency", 5)

	// Logging defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("log_format", "text")
	v.SetDefault("verbose", false)
}

// ValidateConfig validates the configuration settings
func ValidateConfig(config *types.MigrationConfig) error {
	// Check if source directory exists
	if _, err := os.Stat(config.SourceDir); os.IsNotExist(err) {
		return &types.MigrationError{
			Type:    types.ErrorTypeValidation,
			Message: "source directory does not exist: " + config.SourceDir,
		}
	}

	// Validate S3 bucket name if provided
	if config.S3Bucket == "" {
		return &types.MigrationError{
			Type:    types.ErrorTypeValidation,
			Message: "S3 bucket name is required",
		}
	}

	// Validate concurrency
	if config.Concurrency < 1 || config.Concurrency > 20 {
		return &types.MigrationError{
			Type:    types.ErrorTypeValidation,
			Message: "concurrency must be between 1 and 20",
		}
	}

	// Create backup directory if backup is enabled
	if config.BackupOriginals && config.BackupDir != "" {
		if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
			return &types.MigrationError{
				Type:     types.ErrorTypeFileSystem,
				Message:  "failed to create backup directory: " + config.BackupDir,
				Original: err,
			}
		}
	}

	return nil
}

// migrateV1ConfigToV2 handles automatic migration from v0.1.x configuration format to v0.2.0
func migrateV1ConfigToV2(config *types.MigrationConfig) {
	// Check if this is a v0.1.x config by looking for legacy fields and absence of media config
	hasMediaConfig := len(config.MediaTypes) > 0
	hasEnabledMediaTypes := len(config.EnabledMediaTypes) > 0 || len(config.EnabledMediaTypesStr) > 0

	// If already has v0.2.0 media configuration, no migration needed
	if hasMediaConfig || hasEnabledMediaTypes {
		return
	}

	// This appears to be a v0.1.x config - migrate to v0.2.0 format
	// v0.1.x only supported images, so default to images-only mode
	config.EnabledMediaTypesStr = []string{"images"}
	config.EnabledMediaTypes = []types.MediaType{types.MediaTypeImage}

	// Initialize default media configuration with images enabled
	if config.MediaTypes == nil {
		config.MediaTypes = make(map[types.MediaType]*types.MediaConfig)
	}

	// Set up image configuration with default extensions and legacy S3 settings
	config.MediaTypes[types.MediaTypeImage] = &types.MediaConfig{
		Enabled:    true,
		Extensions: []string{"jpg", "jpeg", "png", "gif", "svg", "webp", "ico", "bmp", "tiff", "tif"},
		S3Bucket:   config.S3Bucket, // Inherit from legacy config
		S3Prefix:   config.S3Prefix, // Inherit from legacy config
	}

	// Set other media types as disabled by default
	config.MediaTypes[types.MediaTypeDocument] = &types.MediaConfig{
		Enabled:    false,
		Extensions: []string{"pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx", "txt", "rtf", "odt", "ods", "odp"},
		S3Bucket:   config.S3Bucket, // Inherit from legacy config
		S3Prefix:   "documents/",    // Default prefix for documents
	}
	config.MediaTypes[types.MediaTypeVideo] = &types.MediaConfig{
		Enabled:    false,
		Extensions: []string{"mp4", "avi", "mov", "wmv", "flv", "webm", "mkv", "m4v", "3gp", "ogv"},
		S3Bucket:   config.S3Bucket, // Inherit from legacy config
		S3Prefix:   "videos/",       // Default prefix for videos
	}
	config.MediaTypes[types.MediaTypeAudio] = &types.MediaConfig{
		Enabled:    false,
		Extensions: []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "wma", "opus"},
		S3Bucket:   config.S3Bucket, // Inherit from legacy config
		S3Prefix:   "audio/",        // Default prefix for audio
	}
}
