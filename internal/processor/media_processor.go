package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// MediaProcessor handles validation and processing of various media types
type MediaProcessor struct {
	imageProcessor *ImageProcessor
}

// NewMediaProcessor creates a new media processor
func NewMediaProcessor() *MediaProcessor {
	return &MediaProcessor{
		imageProcessor: NewImageProcessor(),
	}
}

// ValidateMediaFile validates that a file is valid for its media type
func (mp *MediaProcessor) ValidateMediaFile(filePath string, mediaType types.MediaType) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("media file does not exist: %s", filePath)
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot access file %s: %w", filePath, err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filePath)
	}

	// Check file size (prevent empty files)
	if info.Size() == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	// Type-specific validation
	switch mediaType {
	case types.MediaTypeImage:
		return mp.imageProcessor.ValidateImageFile(filePath)
	case types.MediaTypeDocument:
		return mp.validateDocumentFile(filePath)
	case types.MediaTypeVideo:
		return mp.validateVideoFile(filePath)
	case types.MediaTypeAudio:
		return mp.validateAudioFile(filePath)
	case types.MediaTypeOther:
		// Basic validation for other file types
		return nil
	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// validateDocumentFile validates document files
func (mp *MediaProcessor) validateDocumentFile(filePath string) error {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".pdf":
		return mp.validatePDFFile(filePath)
	case ".doc", ".docx":
		return mp.validateWordFile(filePath)
	case ".ppt", ".pptx":
		return mp.validatePowerPointFile(filePath)
	case ".xls", ".xlsx":
		return mp.validateExcelFile(filePath)
	case ".txt", ".rtf":
		// Basic text file validation
		return nil
	default:
		// Unknown document type, basic validation only
		return nil
	}
}

// validateVideoFile validates video files
func (mp *MediaProcessor) validateVideoFile(filePath string) error {
	// Basic video file validation
	// In a real implementation, you might check video headers
	_ = filePath // Parameter reserved for future implementation
	return nil
}

// validateAudioFile validates audio files
func (mp *MediaProcessor) validateAudioFile(filePath string) error {
	// Basic audio file validation
	// In a real implementation, you might check audio headers
	_ = filePath // Parameter reserved for future implementation
	return nil
}

// validatePDFFile validates PDF files
func (mp *MediaProcessor) validatePDFFile(filePath string) error {
	// Basic PDF validation - check for PDF header
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return fmt.Errorf("cannot read PDF header: %w", err)
	}

	if string(header) != "%PDF" {
		return fmt.Errorf("not a valid PDF file: %s", filePath)
	}

	return nil
}

// validateWordFile validates Microsoft Word files
func (mp *MediaProcessor) validateWordFile(filePath string) error {
	// Basic Word document validation
	// For .docx files, we could check if it's a valid ZIP with expected structure
	// For now, just basic file validation
	_ = filePath // Parameter reserved for future implementation
	return nil
}

// validatePowerPointFile validates Microsoft PowerPoint files
func (mp *MediaProcessor) validatePowerPointFile(filePath string) error {
	// Basic PowerPoint validation
	_ = filePath // Parameter reserved for future implementation
	return nil
}

// validateExcelFile validates Microsoft Excel files
func (mp *MediaProcessor) validateExcelFile(filePath string) error {
	// Basic Excel validation
	_ = filePath // Parameter reserved for future implementation
	return nil
}

// GetMediaInfo extracts metadata information from a media file
func (mp *MediaProcessor) GetMediaInfo(filePath string, mediaType types.MediaType) (*MediaInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot access file %s: %w", filePath, err)
	}

	mediaInfo := &MediaInfo{
		FilePath:     filePath,
		MediaType:    mediaType,
		FileSize:     info.Size(),
		LastModified: info.ModTime(),
		ContentType:  mp.getContentType(filePath, mediaType),
	}

	return mediaInfo, nil
}

// getContentType determines the MIME content type for a file
func (mp *MediaProcessor) getContentType(filePath string, mediaType types.MediaType) string {
	ext := filepath.Ext(filePath)

	// Common MIME types based on extension
	mimeTypes := map[string]string{
		// Images
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		".ico":  "image/x-icon",
		".bmp":  "image/bmp",

		// Documents
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".txt":  "text/plain",
		".rtf":  "application/rtf",

		// Videos
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",

		// Audio
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}

	// Default MIME types based on media type
	switch mediaType {
	case types.MediaTypeImage:
		return "image/jpeg"
	case types.MediaTypeDocument:
		return "application/octet-stream"
	case types.MediaTypeVideo:
		return "video/mp4"
	case types.MediaTypeAudio:
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

// MediaInfo contains metadata about a media file
type MediaInfo struct {
	FilePath     string
	MediaType    types.MediaType
	FileSize     int64
	LastModified time.Time
	ContentType  string
}
