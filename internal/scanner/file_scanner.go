package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// FileScanner handles directory traversal and file discovery
type FileScanner struct {
	config            *types.MigrationConfig
	mediaTypeDetector *MediaTypeDetector
	linkParser        *LinkParser
}

// NewFileScanner creates a new file scanner instance
func NewFileScanner(config *types.MigrationConfig) *FileScanner {
	return &FileScanner{
		config:            config,
		mediaTypeDetector: NewMediaTypeDetector(),
		linkParser:        NewLinkParser(),
	}
}

// ScanDirectory recursively scans a directory for files matching the configured extensions
func (fs *FileScanner) ScanDirectory() ([]string, error) {
	var files []string

	err := filepath.Walk(fs.config.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file extension is valid
		if fs.IsValidFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// IsValidFile checks if a file should be processed based on extension and patterns
func (fs *FileScanner) IsValidFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != "" {
		ext = ext[1:] // Remove the dot
	}

	// Check if extension is in the allowed list
	for _, allowedExt := range fs.config.FileExtensions {
		if ext == allowedExt {
			return true
		}
	}

	return false
}

// ScanForMediaReferences scans source files for media references (v0.2.0+)
func (fs *FileScanner) ScanForMediaReferences() ([]types.MediaReference, error) {
	// Get all source files to scan
	sourceFiles, err := fs.ScanDirectory()
	if err != nil {
		return nil, err
	}

	var allReferences []types.MediaReference

	// Process each source file
	for _, filePath := range sourceFiles {
		references, err := fs.linkParser.ParseFileMedia(filePath, fs.config)
		if err != nil {
			// Log error but continue processing other files
			continue
		}
		allReferences = append(allReferences, references...)
	}

	return allReferences, nil
}

// ScanForImageReferences scans source files for image references (legacy method for backward compatibility)
func (fs *FileScanner) ScanForImageReferences() ([]types.ImageReference, error) {
	// Get all source files to scan
	sourceFiles, err := fs.ScanDirectory()
	if err != nil {
		return nil, err
	}

	var allReferences []types.ImageReference

	// Process each source file
	for _, filePath := range sourceFiles {
		references, err := fs.linkParser.ParseFile(filePath)
		if err != nil {
			// Log error but continue processing other files
			continue
		}
		allReferences = append(allReferences, references...)
	}

	return allReferences, nil
}

// ScanMediaFiles scans directory for actual media files based on configured media types
func (fs *FileScanner) ScanMediaFiles() (map[types.MediaType][]string, error) {
	result := make(map[types.MediaType][]string)

	// Initialize result map for all enabled media types
	for _, mediaType := range fs.config.GetEnabledMediaTypes() {
		result[mediaType] = []string{}
	}

	err := filepath.Walk(fs.config.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches any enabled media type
		mediaType := fs.mediaTypeDetector.DetectMediaType(path)
		if fs.config.IsMediaTypeEnabled(mediaType) {
			result[mediaType] = append(result[mediaType], path)
		}

		return nil
	})

	return result, err
}

// IsValidFileForMediaType checks if a file should be processed for a specific media type
func (fs *FileScanner) IsValidFileForMediaType(filePath string, mediaType types.MediaType) bool {
	// Check if media type is enabled
	if !fs.config.IsMediaTypeEnabled(mediaType) {
		return false
	}

	// Check if file extension matches the media type
	detectedType := fs.mediaTypeDetector.DetectMediaType(filePath)
	return detectedType == mediaType
}

// GetSupportedExtensions returns all supported file extensions for enabled media types
func (fs *FileScanner) GetSupportedExtensions() []string {
	var extensions []string

	for _, mediaType := range fs.config.GetEnabledMediaTypes() {
		mediaConfig := fs.config.GetMediaConfig(mediaType)
		extensions = append(extensions, mediaConfig.Extensions...)
	}

	return extensions
}
