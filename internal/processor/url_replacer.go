package processor

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// URLReplacer handles updating source files with new URLs
type URLReplacer struct {
	verbose       bool
	dryRun        bool
	createBackups bool
}

// NewURLReplacer creates a new URL replacer
func NewURLReplacer(verbose, dryRun, createBackups bool) *URLReplacer {
	return &URLReplacer{
		verbose:       verbose,
		dryRun:        dryRun,
		createBackups: createBackups,
	}
}

// ReplaceURLsInFile replaces URLs in a source file
func (ur *URLReplacer) ReplaceURLsInFile(filePath string, replacements map[string]string) error {
	if len(replacements) == 0 {
		return nil
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	originalContent := string(content)
	updatedContent := originalContent

	// Apply replacements
	for oldURL, newURL := range replacements {
		// Escape special regex characters in the old URL
		escapedOldURL := regexp.QuoteMeta(oldURL)

		// Replace in Markdown format: ![...](oldURL)
		markdownPattern := fmt.Sprintf(`(!\[[^\]]*\]\()%s(\))`, escapedOldURL)
		markdownRegex := regexp.MustCompile(markdownPattern)
		updatedContent = markdownRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML format: <img src="oldURL"
		htmlPattern := fmt.Sprintf(`(<img[^>]+src=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlRegex := regexp.MustCompile(htmlPattern)
		updatedContent = htmlRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in YAML frontmatter format: featured_image: "oldURL"
		yamlPattern := fmt.Sprintf(`(?m)(^\s*(?:image|featured_image|hero_image|banner_image|cover_image|thumbnail|avatar):\s*["']?)%s(["']?\s*$)`, escapedOldURL)
		yamlRegex := regexp.MustCompile(yamlPattern)
		updatedContent = yamlRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in Hugo shortcode format: {{< gallery-image src="oldURL" >}}
		hugoPattern := fmt.Sprintf(`(\{\{<\s*(?:gallery-image|figure|img|image)\s+[^>]*(?:src|image)=["'])%s(["'][^>]*>\}\})`, escapedOldURL)
		hugoRegex := regexp.MustCompile(hugoPattern)
		updatedContent = hugoRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in CSS background-image format: background-image: url("oldURL")
		cssPattern := fmt.Sprintf(`(background-image:\s*url\(["']?)%s(["']?\))`, escapedOldURL)
		cssRegex := regexp.MustCompile(cssPattern)
		updatedContent = cssRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		if ur.verbose {
			fmt.Printf("  Replacing: %s -> %s\n", oldURL, newURL)
		}
	}

	// Check if content actually changed
	if updatedContent == originalContent {
		if ur.verbose {
			fmt.Printf("  No changes needed in %s\n", filePath)
		}
		return nil
	}

	if ur.dryRun {
		fmt.Printf("  [DRY RUN] Would update %s\n", filePath)
		return nil
	}

	// Create backup if requested
	if ur.createBackups {
		if err := ur.createBackup(filePath); err != nil {
			return fmt.Errorf("error creating backup for %s: %w", filePath, err)
		}
	}

	// Write updated content
	if err := os.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", filePath, err)
	}

	fmt.Printf("  ✓ Updated %s\n", filePath)
	return nil
}

// createBackup creates a backup of the file with timestamp
func (ur *URLReplacer) createBackup(filePath string) error {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup-%s", filePath, timestamp)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return err
	}

	if ur.verbose {
		fmt.Printf("  Created backup: %s\n", backupPath)
	}
	return nil
}

// UpdateImageReferences updates multiple image references
func (ur *URLReplacer) UpdateImageReferences(refs []types.ImageReference, urlMapping map[string]string) error {
	// Group references by source file
	fileReplacements := make(map[string]map[string]string)

	for _, ref := range refs {
		newURL, exists := urlMapping[ref.OriginalURL]
		if !exists {
			if ur.verbose {
				fmt.Printf("  Warning: No S3 URL mapping found for %s\n", ref.OriginalURL)
			}
			continue
		}

		if fileReplacements[ref.SourceFile] == nil {
			fileReplacements[ref.SourceFile] = make(map[string]string)
		}
		fileReplacements[ref.SourceFile][ref.OriginalURL] = newURL
	}

	// Update each file
	for filePath, replacements := range fileReplacements {
		if ur.verbose {
			fmt.Printf("Updating file: %s\n", filePath)
		}

		if err := ur.ReplaceURLsInFile(filePath, replacements); err != nil {
			return fmt.Errorf("error updating %s: %w", filePath, err)
		}
	}

	return nil
}

// ValidateFileAccess checks if files can be read and written
func (ur *URLReplacer) ValidateFileAccess(refs []types.ImageReference) error {
	checkedFiles := make(map[string]bool)

	for _, ref := range refs {
		if checkedFiles[ref.SourceFile] {
			continue
		}

		// Check if file exists and is readable
		if _, err := os.Stat(ref.SourceFile); err != nil {
			return fmt.Errorf("cannot access file %s: %w", ref.SourceFile, err)
		}

		// Check if file is writable
		if !ur.dryRun {
			file, err := os.OpenFile(ref.SourceFile, os.O_WRONLY, 0)
			if err != nil {
				return fmt.Errorf("cannot write to file %s: %w", ref.SourceFile, err)
			}
			file.Close()
		}

		checkedFiles[ref.SourceFile] = true
	}

	return nil
}
