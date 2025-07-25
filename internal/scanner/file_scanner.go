package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// FileScanner handles directory traversal and file discovery
type FileScanner struct {
	config *types.MigrationConfig
}

// NewFileScanner creates a new file scanner instance
func NewFileScanner(config *types.MigrationConfig) *FileScanner {
	return &FileScanner{
		config: config,
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
} // IsValidFile checks if a file should be processed based on extension and patterns
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
