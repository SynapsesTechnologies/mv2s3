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

// ReplaceURLsInFile replaces URLs in a source file with support for all media types
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

		// Replace in Markdown image format: ![...](oldURL)
		markdownImagePattern := fmt.Sprintf(`(!\[[^\]]*\]\()%s(\))`, escapedOldURL)
		markdownImageRegex := regexp.MustCompile(markdownImagePattern)
		updatedContent = markdownImageRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in Markdown link format: [...](oldURL) - for documents and other media
		markdownLinkPattern := fmt.Sprintf(`(\[[^\]]*\]\()%s(\))`, escapedOldURL)
		markdownLinkRegex := regexp.MustCompile(markdownLinkPattern)
		updatedContent = markdownLinkRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML img format: <img src="oldURL"
		htmlImgPattern := fmt.Sprintf(`(<img[^>]+src=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlImgRegex := regexp.MustCompile(htmlImgPattern)
		updatedContent = htmlImgRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML anchor format: <a href="oldURL"
		htmlAnchorPattern := fmt.Sprintf(`(<a[^>]+href=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlAnchorRegex := regexp.MustCompile(htmlAnchorPattern)
		updatedContent = htmlAnchorRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML embed format: <embed src="oldURL"
		htmlEmbedPattern := fmt.Sprintf(`(<embed[^>]+src=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlEmbedRegex := regexp.MustCompile(htmlEmbedPattern)
		updatedContent = htmlEmbedRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML object format: <object data="oldURL"
		htmlObjectPattern := fmt.Sprintf(`(<object[^>]+data=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlObjectRegex := regexp.MustCompile(htmlObjectPattern)
		updatedContent = htmlObjectRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML iframe format: <iframe src="oldURL"
		htmlIframePattern := fmt.Sprintf(`(<iframe[^>]+src=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlIframeRegex := regexp.MustCompile(htmlIframePattern)
		updatedContent = htmlIframeRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML video format: <video src="oldURL"
		htmlVideoPattern := fmt.Sprintf(`(<video[^>]+src=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlVideoRegex := regexp.MustCompile(htmlVideoPattern)
		updatedContent = htmlVideoRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in HTML audio format: <audio src="oldURL"
		htmlAudioPattern := fmt.Sprintf(`(<audio[^>]+src=["'])%s(["'][^>]*>)`, escapedOldURL)
		htmlAudioRegex := regexp.MustCompile(htmlAudioPattern)
		updatedContent = htmlAudioRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in YAML frontmatter format for images: image: "oldURL"
		yamlImagePattern := fmt.Sprintf(`(?m)(^\s*(?:image|featured_image|hero_image|banner_image|cover_image|thumbnail|avatar):\s*["']?)%s(["']?\s*$)`, escapedOldURL)
		yamlImageRegex := regexp.MustCompile(yamlImagePattern)
		updatedContent = yamlImageRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in YAML frontmatter format for documents: document: "oldURL"
		yamlDocPattern := fmt.Sprintf(`(?m)(^\s*(?:document|download|download_link|file|attachment|pdf_link|doc_link):\s*["']?)%s(["']?\s*$)`, escapedOldURL)
		yamlDocRegex := regexp.MustCompile(yamlDocPattern)
		updatedContent = yamlDocRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in Hugo shortcode format for images: {{< gallery-image src="oldURL" >}}
		hugoImagePattern := fmt.Sprintf(`(\{\{<\s*(?:gallery-image|figure|img|image)\s+[^>]*(?:src|image)=["'])%s(["'][^>]*>\}\})`, escapedOldURL)
		hugoImageRegex := regexp.MustCompile(hugoImagePattern)
		updatedContent = hugoImageRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in Hugo shortcode format for documents: {{< download src="oldURL" >}}
		hugoDocPattern := fmt.Sprintf(`(\{\{<\s*(?:download|document|file|attachment)\s+[^>]*(?:src|href|file|url)=["'])%s(["'][^>]*>\}\})`, escapedOldURL)
		hugoDocRegex := regexp.MustCompile(hugoDocPattern)
		updatedContent = hugoDocRegex.ReplaceAllString(updatedContent, "${1}"+newURL+"${2}")

		// Replace in CSS background format: background-image: url("oldURL") or background: url("oldURL")
		cssPattern := fmt.Sprintf(`((?:background-image|background):\s*[^;]*url\(["']?)%s(["']?\))`, escapedOldURL)
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

// UpdateMediaReferences updates multiple media references (v0.2.0+)
func (ur *URLReplacer) UpdateMediaReferences(refs []types.MediaReference, urlMapping map[string]string) error {
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

// UpdateImageReferences updates multiple image references (legacy method for backward compatibility)
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

// ValidateMediaFileAccess checks if media files can be read and written (v0.2.0+)
func (ur *URLReplacer) ValidateMediaFileAccess(refs []types.MediaReference) error {
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

// ValidateFileAccess checks if files can be read and written (legacy method for backward compatibility)
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
