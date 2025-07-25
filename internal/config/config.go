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
