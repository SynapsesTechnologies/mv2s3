package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// LinkParser handles parsing files for image references
type LinkParser struct {
	// Regex patterns for different image reference formats
	markdownImagePattern *regexp.Regexp
	htmlImagePattern     *regexp.Regexp
	htmlSrcPattern       *regexp.Regexp
	yamlImagePattern     *regexp.Regexp
	hugoShortcodePattern *regexp.Regexp
	cssBackgroundPattern *regexp.Regexp
	inFrontmatter        bool
}

// NewLinkParser creates a new link parser instance
func NewLinkParser() *LinkParser {
	return &LinkParser{
		// Markdown image syntax: ![alt text](path "optional title")
		markdownImagePattern: regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`),
		// HTML img tags: <img src="path" ...>
		htmlImagePattern: regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`),
		// Generic src attribute extraction
		htmlSrcPattern: regexp.MustCompile(`src=["']([^"']+)["']`),
		// YAML frontmatter image fields: image: "path" or featured_image: "path"
		yamlImagePattern: regexp.MustCompile(`(?i)^\s*(?:image|featured_image|hero_image|banner_image|cover_image|thumbnail|avatar):\s*["']?([^"'\s]+)["']?\s*$`),
		// Hugo shortcodes with image parameters: {{< gallery-image src="path" >}} or {{< figure src="path" >}}
		hugoShortcodePattern: regexp.MustCompile(`\{\{<\s*(?:gallery-image|figure|img|image)\s+[^>]*(?:src|image)=["']([^"']+)["'][^>]*>\}\}`),
		// CSS background images: background-image: url("path")
		cssBackgroundPattern: regexp.MustCompile(`background-image:\s*url\(["']?([^"')]+)["']?\)`),
		inFrontmatter:        false,
	}
}

// ParseFile parses a file and extracts image references
func (lp *LinkParser) ParseFile(filePath string) ([]types.ImageReference, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var references []types.ImageReference
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
			refs := lp.parseMarkdownLine(filePath, line, lineNumber)
			references = append(references, refs...)
		case ".html", ".htm":
			refs := lp.parseHTMLLine(filePath, line, lineNumber)
			references = append(references, refs...)
		case ".css":
			refs := lp.parseCSSLine(filePath, line, lineNumber)
			references = append(references, refs...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Filter to only local images (exclude already migrated S3 URLs)
	var localReferences []types.ImageReference
	for _, ref := range references {
		if lp.IsLocalURL(ref.OriginalURL) {
			localReferences = append(localReferences, ref)
		}
	}

	return localReferences, nil
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
	if strings.Contains(url, "://") {
		return false
	}
	// Skip S3 URLs that might not have protocol prefix
	if strings.Contains(url, ".s3.") && strings.Contains(url, ".amazonaws.com") {
		return false
	}
	// Everything else is considered local
	return true
}

// parseMarkdownLine extracts image references from a single line of Markdown
func (lp *LinkParser) parseMarkdownLine(filePath, line string, lineNumber int) []types.ImageReference {
	var references []types.ImageReference

	// Check for YAML frontmatter images if we're in the frontmatter section
	if lp.inFrontmatter {
		yamlRefs := lp.parseYAMLLine(filePath, line, lineNumber)
		references = append(references, yamlRefs...)
	}

	// Find Markdown image syntax: ![alt](path)
	matches := lp.markdownImagePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			altText := match[1]
			imagePath := match[2]

			// Handle titles in image syntax: ![alt](path "title")
			if spaceIndex := strings.Index(imagePath, " "); spaceIndex != -1 {
				imagePath = imagePath[:spaceIndex]
			}

			ref := types.ImageReference{
				SourceFile:  filePath,
				LocalPath:   imagePath, // Will be resolved later
				LineNumber:  lineNumber,
				OriginalURL: imagePath,
			}
			references = append(references, ref)

			// Log for debugging (can be removed later)
			_ = altText // Suppress unused variable warning
		}
	}

	// Check for Hugo shortcodes
	hugoRefs := lp.parseHugoShortcodes(filePath, line, lineNumber)
	references = append(references, hugoRefs...)

	// Also check for HTML img tags within Markdown
	htmlRefs := lp.parseHTMLLine(filePath, line, lineNumber)
	references = append(references, htmlRefs...)

	return references
}

// parseHTMLLine extracts image references from a single line of HTML
func (lp *LinkParser) parseHTMLLine(filePath, line string, lineNumber int) []types.ImageReference {
	var references []types.ImageReference

	// Find HTML img tags
	matches := lp.htmlImagePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			imagePath := match[1]

			ref := types.ImageReference{
				SourceFile:  filePath,
				LocalPath:   imagePath, // Will be resolved later
				LineNumber:  lineNumber,
				OriginalURL: imagePath,
			}
			references = append(references, ref)
		}
	}

	return references
}

// updateFrontmatterState tracks whether we're currently in YAML frontmatter
func (lp *LinkParser) updateFrontmatterState(line string) {
	trimmedLine := strings.TrimSpace(line)

	// YAML frontmatter starts and ends with "---"
	if trimmedLine == "---" {
		lp.inFrontmatter = !lp.inFrontmatter
	}
}

// parseYAMLLine extracts image references from YAML frontmatter
func (lp *LinkParser) parseYAMLLine(filePath, line string, lineNumber int) []types.ImageReference {
	var references []types.ImageReference

	matches := lp.yamlImagePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			imagePath := match[1]

			ref := types.ImageReference{
				SourceFile:  filePath,
				LocalPath:   imagePath, // Will be resolved later
				LineNumber:  lineNumber,
				OriginalURL: imagePath,
			}
			references = append(references, ref)
		}
	}

	return references
}

// parseHugoShortcodes extracts image references from Hugo shortcodes
func (lp *LinkParser) parseHugoShortcodes(filePath, line string, lineNumber int) []types.ImageReference {
	var references []types.ImageReference

	matches := lp.hugoShortcodePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			imagePath := match[1]

			ref := types.ImageReference{
				SourceFile:  filePath,
				LocalPath:   imagePath, // Will be resolved later
				LineNumber:  lineNumber,
				OriginalURL: imagePath,
			}
			references = append(references, ref)
		}
	}

	return references
}

// parseCSSLine extracts image references from CSS background-image properties
func (lp *LinkParser) parseCSSLine(filePath, line string, lineNumber int) []types.ImageReference {
	var references []types.ImageReference

	matches := lp.cssBackgroundPattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			imagePath := match[1]

			ref := types.ImageReference{
				SourceFile:  filePath,
				LocalPath:   imagePath, // Will be resolved later
				LineNumber:  lineNumber,
				OriginalURL: imagePath,
			}
			references = append(references, ref)
		}
	}

	return references
}
