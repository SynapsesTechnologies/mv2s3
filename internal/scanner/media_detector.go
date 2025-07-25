package scanner

import (
	"path/filepath"
	"strings"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// MediaTypeDetector handles detection and classification of media files
type MediaTypeDetector struct {
	// Cache of extension to media type mappings for performance
	extensionCache map[string]types.MediaType
}

// NewMediaTypeDetector creates a new media type detector
func NewMediaTypeDetector() *MediaTypeDetector {
	detector := &MediaTypeDetector{
		extensionCache: make(map[string]types.MediaType),
	}

	// Pre-populate cache with known extensions
	detector.initializeExtensionCache()

	return detector
}

// initializeExtensionCache populates the extension cache with default mappings
func (mtd *MediaTypeDetector) initializeExtensionCache() {
	// Image extensions
	imageExts := []string{"jpg", "jpeg", "png", "gif", "svg", "webp", "ico", "bmp", "tiff", "tif"}
	for _, ext := range imageExts {
		mtd.extensionCache[ext] = types.MediaTypeImage
	}

	// Document extensions
	docExts := []string{"pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx", "txt", "rtf", "odt", "ods", "odp"}
	for _, ext := range docExts {
		mtd.extensionCache[ext] = types.MediaTypeDocument
	}

	// Video extensions
	videoExts := []string{"mp4", "avi", "mov", "wmv", "flv", "webm", "mkv", "m4v", "3gp", "ogv"}
	for _, ext := range videoExts {
		mtd.extensionCache[ext] = types.MediaTypeVideo
	}

	// Audio extensions
	audioExts := []string{"mp3", "wav", "ogg", "flac", "aac", "m4a", "wma", "opus"}
	for _, ext := range audioExts {
		mtd.extensionCache[ext] = types.MediaTypeAudio
	}
}

// DetectMediaType determines the media type of a file based on its extension
func (mtd *MediaTypeDetector) DetectMediaType(filePath string) types.MediaType {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != "" && len(ext) > 1 {
		ext = ext[1:] // Remove the dot prefix
	}

	if mediaType, exists := mtd.extensionCache[ext]; exists {
		return mediaType
	}

	return types.MediaTypeOther
}

// IsMediaTypeEnabled checks if a media type is enabled in the configuration
func (mtd *MediaTypeDetector) IsMediaTypeEnabled(mediaType types.MediaType, config *types.MigrationConfig) bool {
	return config.IsMediaTypeEnabled(mediaType)
}

// IsFileSupported checks if a file is supported based on media type and configuration
func (mtd *MediaTypeDetector) IsFileSupported(filePath string, config *types.MigrationConfig) bool {
	mediaType := mtd.DetectMediaType(filePath)
	return mtd.IsMediaTypeEnabled(mediaType, config)
}

// GetSupportedExtensions returns all supported extensions for enabled media types
func (mtd *MediaTypeDetector) GetSupportedExtensions(config *types.MigrationConfig) []string {
	var extensions []string
	extensionSet := make(map[string]bool) // To avoid duplicates

	enabledTypes := config.GetEnabledMediaTypes()
	for _, mediaType := range enabledTypes {
		mediaConfig := config.GetMediaConfig(mediaType)
		for _, ext := range mediaConfig.Extensions {
			if !extensionSet[ext] {
				extensions = append(extensions, ext)
				extensionSet[ext] = true
			}
		}
	}

	return extensions
}

// GetMediaTypeStats returns statistics about media types in a file list
func (mtd *MediaTypeDetector) GetMediaTypeStats(filePaths []string, config *types.MigrationConfig) map[types.MediaType]int {
	stats := make(map[types.MediaType]int)

	for _, filePath := range filePaths {
		mediaType := mtd.DetectMediaType(filePath)
		if mtd.IsMediaTypeEnabled(mediaType, config) {
			stats[mediaType]++
		}
	}

	return stats
}

// FilterFilesByMediaType filters a list of files by media type
func (mtd *MediaTypeDetector) FilterFilesByMediaType(filePaths []string, mediaType types.MediaType) []string {
	var filtered []string

	for _, filePath := range filePaths {
		if mtd.DetectMediaType(filePath) == mediaType {
			filtered = append(filtered, filePath)
		}
	}

	return filtered
}

// UpdateExtensionCache allows adding custom extension mappings
func (mtd *MediaTypeDetector) UpdateExtensionCache(extensions map[string]types.MediaType) {
	for ext, mediaType := range extensions {
		ext = strings.ToLower(ext)
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:] // Remove dot prefix if present
		}
		mtd.extensionCache[ext] = mediaType
	}
}

// GetExtensionMapping returns the current extension to media type mapping
func (mtd *MediaTypeDetector) GetExtensionMapping() map[string]types.MediaType {
	// Return a copy to prevent external modification
	mapping := make(map[string]types.MediaType)
	for ext, mediaType := range mtd.extensionCache {
		mapping[ext] = mediaType
	}
	return mapping
}
