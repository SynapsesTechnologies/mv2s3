package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// LinkParser handles parsing files for media references (images, documents, etc.)
type LinkParser struct {
	// Image-specific patterns
	markdownImagePattern *regexp.Regexp
	htmlImagePattern     *regexp.Regexp
	htmlSrcPattern       *regexp.Regexp
	yamlImagePattern     *regexp.Regexp
	hugoShortcodePattern *regexp.Regexp
	cssBackgroundPattern *regexp.Regexp

	// Document-specific patterns
	markdownLinkPattern     *regexp.Regexp // [text](document.pdf)
	htmlAnchorPattern       *regexp.Regexp // <a href="document.pdf">
	htmlEmbedPattern        *regexp.Regexp // <embed src="document.pdf">
	htmlObjectPattern       *regexp.Regexp // <object data="document.pdf">
	htmlIframePattern       *regexp.Regexp // <iframe src="document.pdf">
	yamlDocumentPattern     *regexp.Regexp // document: "path.pdf" or download_link: "path.pdf"
	hugoDocShortcodePattern *regexp.Regexp // {{< download src="doc.pdf" >}}

	// State tracking
	inFrontmatter     bool
	mediaTypeDetector *MediaTypeDetector
}

// NewLinkParser creates a new link parser instance
func NewLinkParser() *LinkParser {
	return &LinkParser{
		// Image patterns
		markdownImagePattern: regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`),
		htmlImagePattern:     regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`),
		htmlSrcPattern:       regexp.MustCompile(`src=["']([^"']+)["']`),
		yamlImagePattern:     regexp.MustCompile(`(?i)^\s*(?:image|featured_image|hero_image|banner_image|cover_image|thumbnail|avatar):\s*["']?([^"'\s]+)["']?\s*$`),
		hugoShortcodePattern: regexp.MustCompile(`\{\{<\s*(?:gallery-image|figure|img|image)\s+[^>]*(?:src|image)=["']([^"']+)["'][^>]*>\}\}`),
		cssBackgroundPattern: regexp.MustCompile(`(?:background-image|background):\s*[^;]*url\(["']?([^"')]+)["']?\)`),

		// Document patterns
		markdownLinkPattern:     regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`), // [text](link)
		htmlAnchorPattern:       regexp.MustCompile(`<a[^>]+href=["']([^"']+)["'][^>]*>`),
		htmlEmbedPattern:        regexp.MustCompile(`<embed[^>]+src=["']([^"']+)["'][^>]*>`),
		htmlObjectPattern:       regexp.MustCompile(`<object[^>]+data=["']([^"']+)["'][^>]*>`),
		htmlIframePattern:       regexp.MustCompile(`<iframe[^>]+src=["']([^"']+)["'][^>]*>`),
		yamlDocumentPattern:     regexp.MustCompile(`(?i)^\s*(?:document|download|download_link|file|attachment|pdf_link|doc_link):\s*["']?([^"'\s]+)["']?\s*$`),
		hugoDocShortcodePattern: regexp.MustCompile(`\{\{<\s*(?:download|document|file|attachment)\s+[^>]*(?:src|href|file|url)=["']([^"']+)["'][^>]*>\}\}`),

		// Initialize components
		inFrontmatter:     false,
		mediaTypeDetector: NewMediaTypeDetector(),
	}
}

// ParseFileMedia parses a file and extracts media references (new method for v0.2.0+)
func (lp *LinkParser) ParseFileMedia(filePath string, config *types.MigrationConfig) ([]types.MediaReference, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var references []types.MediaReference
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Track frontmatter boundaries for Markdown files
		if strings.ToLower(filepath.Ext(filePath)) == ".md" || strings.ToLower(filepath.Ext(filePath)) == ".markdown" {
			lp.updateFrontmatterState(line)
		}

		// Parse based on file extension
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".md", ".markdown":
			refs := lp.parseMarkdownLineMedia(filePath, line, lineNumber, config)
			references = append(references, refs...)
		case ".html", ".htm":
			refs := lp.parseHTMLLineMedia(filePath, line, lineNumber, config)
			references = append(references, refs...)
		case ".css":
			refs := lp.parseCSSLineMedia(filePath, line, lineNumber, config)
			references = append(references, refs...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Filter to only local media files (exclude already migrated S3 URLs)
	var localReferences []types.MediaReference
	for _, ref := range references {
		if lp.IsLocalURL(ref.OriginalURL) {
			localReferences = append(localReferences, ref)
		}
	}

	return localReferences, nil
}

// ParseFile parses a file and extracts image references (backward compatibility)
func (lp *LinkParser) ParseFile(filePath string) ([]types.ImageReference, error) {
	// Create a minimal config for backward compatibility (images only)
	config := &types.MigrationConfig{}
	config.InitializeMediaTypes()
	config.EnabledMediaTypes = []types.MediaType{types.MediaTypeImage}

	mediaRefs, err := lp.ParseFileMedia(filePath, config)
	if err != nil {
		return nil, err
	}

	// Convert MediaReference to ImageReference for backward compatibility
	var imageRefs []types.ImageReference
	for _, mediaRef := range mediaRefs {
		if mediaRef.MediaType == types.MediaTypeImage {
			imageRefs = append(imageRefs, types.ImageReference{
				SourceFile:   mediaRef.SourceFile,
				LocalPath:    mediaRef.LocalPath,
				LineNumber:   mediaRef.LineNumber,
				OriginalURL:  mediaRef.OriginalURL,
				NewURL:       mediaRef.NewURL,
				FileSize:     mediaRef.FileSize,
				ContentType:  mediaRef.ContentType,
				LastModified: mediaRef.LastModified,
			})
		}
	}

	return imageRefs, nil
}

// IsS3URL determines if a URL is an S3 URL (already migrated)
func (lp *LinkParser) IsS3URL(url string) bool {
	return (strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) &&
		strings.Contains(url, ".s3.") &&
		strings.Contains(url, ".amazonaws.com")
}

// IsLocalURL determines if a URL refers to a local file
func (lp *LinkParser) IsLocalURL(url string) bool {
	// Skip external URLs (including S3 URLs)
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "//") {
		return false
	}
	// Skip data URLs
	if strings.HasPrefix(url, "data:") {
		return false
	}
	// Skip mailto and other protocols
	if strings.Contains(url, "://") || strings.HasPrefix(url, "mailto:") {
		return false
	}
	// Skip S3 URLs that might not have protocol prefix
	if strings.Contains(url, ".s3.") && strings.Contains(url, ".amazonaws.com") {
		return false
	}
	// Everything else is considered local
	return true
}

// Media-aware parsing methods (v0.2.0+)

// updateFrontmatterState tracks whether we're currently in YAML frontmatter
func (lp *LinkParser) updateFrontmatterState(line string) {
	trimmedLine := strings.TrimSpace(line)

	// YAML frontmatter starts and ends with "---"
	if trimmedLine == "---" {
		lp.inFrontmatter = !lp.inFrontmatter
	}
}

// parseMarkdownLineMedia extracts media references from a single line of Markdown
func (lp *LinkParser) parseMarkdownLineMedia(filePath, line string, lineNumber int, config *types.MigrationConfig) []types.MediaReference {
	var references []types.MediaReference

	// Check for YAML frontmatter if we're in the frontmatter section
	if lp.inFrontmatter {
		yamlRefs := lp.parseYAMLLineMedia(filePath, line, lineNumber, config)
		references = append(references, yamlRefs...)
	}

	// Find Markdown image syntax: ![alt](path)
	if config.IsMediaTypeEnabled(types.MediaTypeImage) {
		matches := lp.markdownImagePattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				imagePath := match[2]
				// Handle titles in image syntax: ![alt](path "title")
				if spaceIndex := strings.Index(imagePath, " "); spaceIndex != -1 {
					imagePath = imagePath[:spaceIndex]
				}

				if mediaType := lp.mediaTypeDetector.DetectMediaType(imagePath); mediaType == types.MediaTypeImage {
					ref := types.MediaReference{
						SourceFile:  filePath,
						LocalPath:   imagePath,
						LineNumber:  lineNumber,
						OriginalURL: imagePath,
						MediaType:   mediaType,
					}
					references = append(references, ref)
				}
			}
		}
	}

	// Find Markdown link syntax: [text](path) - for documents
	if config.IsMediaTypeEnabled(types.MediaTypeDocument) || config.IsMediaTypeEnabled(types.MediaTypeVideo) || config.IsMediaTypeEnabled(types.MediaTypeAudio) {
		matches := lp.markdownLinkPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				linkPath := match[2]
				// Handle titles in link syntax: [text](path "title")
				if spaceIndex := strings.Index(linkPath, " "); spaceIndex != -1 {
					linkPath = linkPath[:spaceIndex]
				}

				mediaType := lp.mediaTypeDetector.DetectMediaType(linkPath)
				if config.IsMediaTypeEnabled(mediaType) && mediaType != types.MediaTypeImage {
					ref := types.MediaReference{
						SourceFile:  filePath,
						LocalPath:   linkPath,
						LineNumber:  lineNumber,
						OriginalURL: linkPath,
						MediaType:   mediaType,
					}
					references = append(references, ref)
				}
			}
		}
	}

	// Check for Hugo shortcodes
	hugoRefs := lp.parseHugoShortcodesMedia(filePath, line, lineNumber, config)
	references = append(references, hugoRefs...)

	// Also check for HTML tags within Markdown
	htmlRefs := lp.parseHTMLLineMedia(filePath, line, lineNumber, config)
	references = append(references, htmlRefs...)

	return references
}

// parseHTMLLineMedia extracts media references from a single line of HTML
func (lp *LinkParser) parseHTMLLineMedia(filePath, line string, lineNumber int, config *types.MigrationConfig) []types.MediaReference {
	var references []types.MediaReference

	// Find HTML img tags
	if config.IsMediaTypeEnabled(types.MediaTypeImage) {
		matches := lp.htmlImagePattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				imagePath := match[1]
				if mediaType := lp.mediaTypeDetector.DetectMediaType(imagePath); mediaType == types.MediaTypeImage {
					ref := types.MediaReference{
						SourceFile:  filePath,
						LocalPath:   imagePath,
						LineNumber:  lineNumber,
						OriginalURL: imagePath,
						MediaType:   mediaType,
					}
					references = append(references, ref)
				}
			}
		}
	}

	// Find HTML anchor tags (for documents)
	matches := lp.htmlAnchorPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			linkPath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(linkPath)
			if config.IsMediaTypeEnabled(mediaType) && mediaType != types.MediaTypeImage {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   linkPath,
					LineNumber:  lineNumber,
					OriginalURL: linkPath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	// Find HTML embed tags
	matches = lp.htmlEmbedPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			embedPath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(embedPath)
			if config.IsMediaTypeEnabled(mediaType) {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   embedPath,
					LineNumber:  lineNumber,
					OriginalURL: embedPath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	// Find HTML object tags
	matches = lp.htmlObjectPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			objectPath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(objectPath)
			if config.IsMediaTypeEnabled(mediaType) {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   objectPath,
					LineNumber:  lineNumber,
					OriginalURL: objectPath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	// Find HTML iframe tags
	matches = lp.htmlIframePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			iframePath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(iframePath)
			if config.IsMediaTypeEnabled(mediaType) {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   iframePath,
					LineNumber:  lineNumber,
					OriginalURL: iframePath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	return references
}

// parseCSSLineMedia extracts media references from CSS
func (lp *LinkParser) parseCSSLineMedia(filePath, line string, lineNumber int, config *types.MigrationConfig) []types.MediaReference {
	var references []types.MediaReference

	// Find CSS background-image properties
	matches := lp.cssBackgroundPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			imagePath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(imagePath)
			if config.IsMediaTypeEnabled(mediaType) {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   imagePath,
					LineNumber:  lineNumber,
					OriginalURL: imagePath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	return references
}

// parseYAMLLineMedia extracts media references from YAML frontmatter
func (lp *LinkParser) parseYAMLLineMedia(filePath, line string, lineNumber int, config *types.MigrationConfig) []types.MediaReference {
	var references []types.MediaReference

	// Check for image fields
	if config.IsMediaTypeEnabled(types.MediaTypeImage) {
		matches := lp.yamlImagePattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				imagePath := match[1]
				if mediaType := lp.mediaTypeDetector.DetectMediaType(imagePath); mediaType == types.MediaTypeImage {
					ref := types.MediaReference{
						SourceFile:  filePath,
						LocalPath:   imagePath,
						LineNumber:  lineNumber,
						OriginalURL: imagePath,
						MediaType:   mediaType,
					}
					references = append(references, ref)
				}
			}
		}
	}

	// Check for document fields
	matches := lp.yamlDocumentPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			docPath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(docPath)
			if config.IsMediaTypeEnabled(mediaType) {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   docPath,
					LineNumber:  lineNumber,
					OriginalURL: docPath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	return references
}

// parseHugoShortcodesMedia extracts media references from Hugo shortcodes
func (lp *LinkParser) parseHugoShortcodesMedia(filePath, line string, lineNumber int, config *types.MigrationConfig) []types.MediaReference {
	var references []types.MediaReference

	// Check for image shortcodes
	if config.IsMediaTypeEnabled(types.MediaTypeImage) {
		matches := lp.hugoShortcodePattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				imagePath := match[1]
				if mediaType := lp.mediaTypeDetector.DetectMediaType(imagePath); mediaType == types.MediaTypeImage {
					ref := types.MediaReference{
						SourceFile:  filePath,
						LocalPath:   imagePath,
						LineNumber:  lineNumber,
						OriginalURL: imagePath,
						MediaType:   mediaType,
					}
					references = append(references, ref)
				}
			}
		}
	}

	// Check for document shortcodes
	matches := lp.hugoDocShortcodePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			docPath := match[1]
			mediaType := lp.mediaTypeDetector.DetectMediaType(docPath)
			if config.IsMediaTypeEnabled(mediaType) {
				ref := types.MediaReference{
					SourceFile:  filePath,
					LocalPath:   docPath,
					LineNumber:  lineNumber,
					OriginalURL: docPath,
					MediaType:   mediaType,
				}
				references = append(references, ref)
			}
		}
	}

	return references
}
