package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SynapsesTechnologies/mv2s3/internal/config"
	"github.com/SynapsesTechnologies/mv2s3/internal/processor"
	"github.com/SynapsesTechnologies/mv2s3/internal/scanner"
	"github.com/SynapsesTechnologies/mv2s3/internal/storage"
	"github.com/SynapsesTechnologies/mv2s3/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	cfgFile string
	verbose bool
)

// ScanResults represents the output of a scan operation
type ScanResults struct {
	Timestamp       time.Time              `json:"timestamp"`
	SourceDir       string                 `json:"source_dir"`
	Extensions      []string               `json:"extensions"`
	TotalFiles      int                    `json:"total_files"`
	TotalMedia      int                    `json:"total_media"`
	TotalImages     int                    `json:"total_images"` // Backward compatibility
	MediaReferences []types.MediaReference `json:"media_references"`
	References      []types.ImageReference `json:"references"`  // Backward compatibility
	MediaStats      map[string]int         `json:"media_stats"` // Stats by media type
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mv2s3",
	Short: "Move local media files to S3 and update references",
	Long: `mv2s3 is a command-line tool that scans web source code files for local media references,
uploads those media files (images, documents, videos, audio) to AWS S3 (or compatible storage), 
and updates the source code with new URLs.

Supports multiple media types:
- Images: jpg, png, gif, svg, webp, ico, bmp, tiff
- Documents: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, odt, ods, odp
- Videos: mp4, avi, mov, wmv, flv, webm, mkv, m4v, 3gp, ogv
- Audio: mp3, wav, ogg, flac, aac, m4a, wma, opus`,
	Version: "0.2.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mv2s3.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	// Configuration initialization will be implemented later
	if verbose {
		fmt.Println("Verbose mode enabled")
	}
}

// getUniqueFiles returns unique source file paths from image references
func getUniqueFiles(refs []types.ImageReference) []string {
	fileSet := make(map[string]bool)
	for _, ref := range refs {
		fileSet[ref.SourceFile] = true
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}

// cleanS3Key cleans up the S3 key by removing relative path prefixes and normalizing slashes
func cleanS3Key(originalURL, prefix string) string {
	// Start with original URL
	key := originalURL

	// Remove common relative prefixes
	key = strings.TrimPrefix(key, "./")
	key = strings.TrimPrefix(key, "../")
	key = strings.TrimPrefix(key, "/")

	// Add prefix if specified
	if prefix != "" {
		cleanPrefix := strings.TrimSuffix(prefix, "/")
		key = cleanPrefix + "/" + key
	}

	// Normalize path separators for consistency
	key = filepath.ToSlash(key)

	return key
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// mergeConfigWithFlags loads configuration and merges it with command flags
// Command flags take precedence over configuration file values
func mergeConfigWithFlags(cmd *cobra.Command) (*types.MigrationConfig, error) {
	// Load configuration from file/environment/defaults
	cfg, _, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override config values with command flags (flags take precedence)
	if cmd.Flags().Changed("source") {
		cfg.SourceDir, _ = cmd.Flags().GetString("source")
	}
	if cmd.Flags().Changed("extensions") {
		cfg.FileExtensions, _ = cmd.Flags().GetStringSlice("extensions")
	}
	if cmd.Flags().Changed("bucket") {
		cfg.S3Bucket, _ = cmd.Flags().GetString("bucket")
	}
	if cmd.Flags().Changed("region") {
		cfg.S3Region, _ = cmd.Flags().GetString("region")
	}
	if cmd.Flags().Changed("prefix") {
		cfg.S3Prefix, _ = cmd.Flags().GetString("prefix")
	}
	if cmd.Flags().Changed("dry-run") {
		cfg.DryRun, _ = cmd.Flags().GetBool("dry-run")
	}
	if cmd.Flags().Changed("cleanup") {
		cfg.CleanupLocal, _ = cmd.Flags().GetBool("cleanup")
	}
	if cmd.Flags().Changed("backup") {
		cfg.BackupOriginals, _ = cmd.Flags().GetBool("backup")
	}
	if cmd.Flags().Changed("backup-dir") {
		cfg.BackupDir, _ = cmd.Flags().GetString("backup-dir")
	}
	if cmd.Flags().Changed("verbose") {
		cfg.Verbose, _ = cmd.Flags().GetBool("verbose")
	}

	// Override global verbose flag
	if verbose {
		cfg.Verbose = true
	}

	// Handle media type flags (v0.2.0+)
	if err := handleMediaTypeFlags(cmd, cfg); err != nil {
		return nil, fmt.Errorf("failed to process media type flags: %w", err)
	}

	return cfg, nil
}

// handleMediaTypeFlags processes media type-related command flags and updates the configuration
func handleMediaTypeFlags(cmd *cobra.Command, cfg *types.MigrationConfig) error {
	// Initialize MediaTypes map if not already initialized
	if cfg.MediaTypes == nil {
		cfg.MediaTypes = make(map[types.MediaType]*types.MediaConfig)
	}

	// Check if --media-types flag was used
	if cmd.Flags().Changed("media-types") {
		mediaTypesStr, _ := cmd.Flags().GetStringSlice("media-types")
		cfg.EnabledMediaTypesStr = mediaTypesStr

		// Parse to MediaType enums
		cfg.EnabledMediaTypes = make([]types.MediaType, 0, len(mediaTypesStr))
		for _, typeStr := range mediaTypesStr {
			mediaType, err := types.ParseMediaType(typeStr)
			if err != nil {
				return fmt.Errorf("invalid media type '%s': %w", typeStr, err)
			}
			cfg.EnabledMediaTypes = append(cfg.EnabledMediaTypes, mediaType)
		}
	}

	// Handle individual media type flags (these override --media-types)
	enabledTypes := make([]types.MediaType, 0, 4)

	if cmd.Flags().Changed("images") {
		enabled, _ := cmd.Flags().GetBool("images")
		if enabled {
			enabledTypes = append(enabledTypes, types.MediaTypeImage)
		}
	}

	if cmd.Flags().Changed("documents") {
		enabled, _ := cmd.Flags().GetBool("documents")
		if enabled {
			enabledTypes = append(enabledTypes, types.MediaTypeDocument)
		}
	}

	if cmd.Flags().Changed("videos") {
		enabled, _ := cmd.Flags().GetBool("videos")
		if enabled {
			enabledTypes = append(enabledTypes, types.MediaTypeVideo)
		}
	}

	if cmd.Flags().Changed("audio") {
		enabled, _ := cmd.Flags().GetBool("audio")
		if enabled {
			enabledTypes = append(enabledTypes, types.MediaTypeAudio)
		}
	}

	// If individual flags were used, override the media types configuration
	if len(enabledTypes) > 0 {
		cfg.EnabledMediaTypes = enabledTypes
		cfg.EnabledMediaTypesStr = make([]string, len(enabledTypes))
		for i, mt := range enabledTypes {
			cfg.EnabledMediaTypesStr[i] = mt.String()
		}
	}

	// Handle include-extensions flag
	if cmd.Flags().Changed("include-extensions") {
		includeExts, _ := cmd.Flags().GetStringSlice("include-extensions")
		// Add these extensions to the configuration for the "other" media type
		if len(includeExts) > 0 {
			if cfg.MediaTypes[types.MediaTypeOther] == nil {
				cfg.MediaTypes[types.MediaTypeOther] = &types.MediaConfig{
					Enabled:    true,
					Extensions: includeExts,
					S3Bucket:   cfg.S3Bucket,
					S3Prefix:   "other/",
				}
			} else {
				cfg.MediaTypes[types.MediaTypeOther].Extensions = append(
					cfg.MediaTypes[types.MediaTypeOther].Extensions,
					includeExts...,
				)
			}

			// Add "other" to enabled types if not already present
			found := false
			for _, mt := range cfg.EnabledMediaTypes {
				if mt == types.MediaTypeOther {
					found = true
					break
				}
			}
			if !found {
				cfg.EnabledMediaTypes = append(cfg.EnabledMediaTypes, types.MediaTypeOther)
				cfg.EnabledMediaTypesStr = append(cfg.EnabledMediaTypesStr, "other")
			}
		}
	}

	return nil
}

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate media files from local storage to S3",
	Long: `Migrate media files (images, documents, videos, audio) from local storage to S3 and update source code references.

Supports multiple media types:
- Images: jpg, png, gif, svg, webp, ico, bmp, tiff (enabled by default)
- Documents: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, odt, ods, odp
- Videos: mp4, avi, mov, wmv, flv, webm, mkv, m4v, 3gp, ogv  
- Audio: mp3, wav, ogg, flac, aac, m4a, wma, opus

Use --media-types to specify which types to migrate, or use individual flags like --documents --videos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration merged with command flags
		cfg, err := mergeConfigWithFlags(cmd)
		if err != nil {
			return err
		}

		// Get remaining command-specific flags
		scanFile, _ := cmd.Flags().GetString("scan-file")

		fmt.Printf("Starting migration...\n")
		fmt.Printf("S3 Bucket: %s\n", cfg.S3Bucket)
		fmt.Printf("Region: %s\n", cfg.S3Region)
		if cfg.S3Prefix != "" {
			fmt.Printf("Prefix: %s\n", cfg.S3Prefix)
		}
		if cfg.DryRun {
			fmt.Printf("DRY RUN MODE - No changes will be made\n")
		}
		fmt.Println()

		// Validate required configuration
		if cfg.S3Bucket == "" {
			return fmt.Errorf("S3 bucket is required (use --bucket flag or set s3_bucket in config)")
		}

		var allReferences []types.ImageReference
		var totalFiles int

		if scanFile != "" {
			// Load from scan file
			fmt.Printf("Loading scan results from: %s\n", scanFile)
			results, err := loadScanResults(scanFile)
			if err != nil {
				return fmt.Errorf("error loading scan file: %w", err)
			}

			allReferences = results.References
			totalFiles = results.TotalFiles
			fmt.Printf("Loaded %d image references from %d files\n", len(allReferences), totalFiles)
		} else {
			// Perform live scan
			fmt.Printf("Scanning directory: %s\n", cfg.SourceDir)
			fmt.Printf("File extensions: %v\n", cfg.FileExtensions)

			config := &types.MigrationConfig{
				SourceDir:      cfg.SourceDir,
				FileExtensions: cfg.FileExtensions,
				Verbose:        cfg.Verbose,
			}

			// Reuse scan logic
			fileScanner := scanner.NewFileScanner(config)
			files, err := fileScanner.ScanDirectory()
			if err != nil {
				return fmt.Errorf("error scanning directory: %w", err)
			}

			linkParser := scanner.NewLinkParser()
			for _, file := range files {
				references, err := linkParser.ParseFile(file)
				if err != nil {
					fmt.Printf("✗ %s (error: %v)\n", file, err)
					continue
				}
				allReferences = append(allReferences, references...)
			}

			totalFiles = len(files)
			fmt.Printf("Found %d image references in %d files\n", len(allReferences), totalFiles)
		}

		if len(allReferences) == 0 {
			fmt.Println("No images found to migrate.")
			return nil
		}

		fmt.Printf("\nMigration Plan:\n")
		for i, ref := range allReferences {
			s3Key := cleanS3Key(ref.OriginalURL, cfg.S3Prefix)
			s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.S3Bucket, cfg.S3Region, s3Key)

			fmt.Printf("%d. %s:%d\n", i+1, ref.SourceFile, ref.LineNumber)
			fmt.Printf("   %s -> %s\n", ref.OriginalURL, s3URL)
		}

		fmt.Printf("\nSummary:\n")
		fmt.Printf("- Files to process: %d\n", totalFiles)
		fmt.Printf("- Images to migrate: %d\n", len(allReferences))
		fmt.Printf("- S3 Bucket: %s\n", cfg.S3Bucket)

		if cfg.DryRun {
			fmt.Println("\n✓ Dry run completed - no changes made")
			return nil
		}

		// Start actual migration
		fmt.Println("\nStarting migration process...")

		// 1. Initialize S3 client
		fmt.Println("1. Initializing S3 client...")
		migrationConfig := &types.MigrationConfig{
			S3Bucket: cfg.S3Bucket,
			S3Region: cfg.S3Region,
			S3Prefix: cfg.S3Prefix,
		}
		s3Client, err := storage.NewS3Client(migrationConfig)
		if err != nil {
			return fmt.Errorf("error creating S3 client: %w", err)
		}

		// 2. Verify bucket exists
		fmt.Printf("2. Verifying S3 bucket '%s' exists...\n", cfg.S3Bucket)
		exists, err := s3Client.BucketExists(cfg.S3Bucket)
		if err != nil {
			return fmt.Errorf("error checking bucket existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("bucket '%s' does not exist", cfg.S3Bucket)
		}

		// 3. Upload images and build URL mapping
		fmt.Println("3. Uploading images to S3...")
		urlMapping := make(map[string]string)
		uploadedCount := 0
		skippedCount := 0
		failedCount := 0

		for i, ref := range allReferences {
			// Construct S3 key using the same logic as preview
			s3Key := cleanS3Key(ref.OriginalURL, cfg.S3Prefix)

			// Construct full path to source image
			var imagePath string
			if filepath.IsAbs(ref.OriginalURL) {
				imagePath = ref.OriginalURL
			} else {
				// Resolve relative to source file's directory
				sourceDir := filepath.Dir(ref.SourceFile)
				imagePath = filepath.Join(sourceDir, ref.OriginalURL)
			}

			fmt.Printf("   [%d/%d] Processing %s -> s3://%s/%s\n", i+1, len(allReferences), ref.OriginalURL, cfg.S3Bucket, s3Key)

			// Check if image file exists
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				fmt.Printf("   ✗ Image file not found: %s\n", imagePath)
				failedCount++
				continue
			}

			// Check if object already exists in S3
			exists, err := s3Client.ObjectExists(cfg.S3Bucket, s3Key)
			if err != nil {
				fmt.Printf("   ✗ Error checking S3 object existence: %v\n", err)
				failedCount++
				continue
			}

			if exists {
				// Object already exists, skip upload but add to URL mapping
				s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.S3Bucket, cfg.S3Region, s3Key)
				urlMapping[ref.OriginalURL] = s3URL
				skippedCount++
				fmt.Printf("   ↺ Already exists in S3, skipping upload\n")
				continue
			}

			// Upload to S3
			s3URL, err := s3Client.UploadFile(imagePath, cfg.S3Bucket, s3Key)
			if err != nil {
				fmt.Printf("   ✗ Upload failed: %v\n", err)
				failedCount++
				continue
			}

			urlMapping[ref.OriginalURL] = s3URL
			uploadedCount++
			if cfg.Verbose {
				fmt.Printf("   ✓ Uploaded: %s\n", s3URL)
			}
		}

		fmt.Printf("\nUpload Summary:\n")
		fmt.Printf("- Successfully uploaded: %d\n", uploadedCount)
		fmt.Printf("- Already existed (skipped): %d\n", skippedCount)
		fmt.Printf("- Failed uploads: %d\n", failedCount)

		if uploadedCount == 0 && skippedCount == 0 {
			return fmt.Errorf("no images were successfully processed")
		}

		// 4. Update source files with new URLs
		fmt.Println("\n4. Updating source files with S3 URLs...")
		urlReplacer := processor.NewURLReplacer(cfg.Verbose, false, cfg.BackupOriginals) // Use config for backups

		// Validate file access first
		if err := urlReplacer.ValidateFileAccess(allReferences); err != nil {
			return fmt.Errorf("file access validation failed: %w", err)
		}

		// Update files
		if err := urlReplacer.UpdateImageReferences(allReferences, urlMapping); err != nil {
			return fmt.Errorf("error updating source files: %w", err)
		}

		// 5. Success summary
		fmt.Printf("\n✓ Migration completed successfully!\n")
		fmt.Printf("Summary:\n")
		fmt.Printf("- Images uploaded to S3: %d\n", uploadedCount)
		fmt.Printf("- Images already in S3: %d\n", skippedCount)
		fmt.Printf("- Source files updated: %d\n", len(getUniqueFiles(allReferences)))
		fmt.Printf("- S3 bucket: %s\n", cfg.S3Bucket)
		if cfg.S3Prefix != "" {
			fmt.Printf("- S3 prefix: %s\n", cfg.S3Prefix)
		}
		if cfg.BackupOriginals {
			fmt.Printf("- Backup files created for safety\n")
		}

		// 6. Optional cleanup of local files
		if cfg.CleanupLocal {
			fmt.Printf("\n6. Cleaning up local image files...\n")

			// Only clean up files that were successfully uploaded
			var successfulRefs []types.ImageReference
			for _, ref := range allReferences {
				if _, exists := urlMapping[ref.OriginalURL]; exists {
					successfulRefs = append(successfulRefs, ref)
				}
			}

			deletedCount := 0
			failedCount := 0

			for i, ref := range successfulRefs {
				// Construct full path to source image
				var imagePath string
				if filepath.IsAbs(ref.OriginalURL) {
					imagePath = ref.OriginalURL
				} else {
					// Resolve relative to source file's directory
					sourceDir := filepath.Dir(ref.SourceFile)
					imagePath = filepath.Join(sourceDir, ref.OriginalURL)
				}

				fmt.Printf("   [%d/%d] Deleting %s", i+1, len(successfulRefs), imagePath)

				// Delete file
				if err := os.Remove(imagePath); err != nil {
					fmt.Printf(" ✗ (failed: %v)\n", err)
					failedCount++
				} else {
					fmt.Printf(" ✓\n")
					deletedCount++
				}
			}

			fmt.Printf("\nCleanup Summary:\n")
			fmt.Printf("- Local files deleted: %d\n", deletedCount)
			fmt.Printf("- Failed deletions: %d\n", failedCount)
		}

		return nil
	},
}

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan source files for media references",
	Long: `Scan source files for media references without making changes (dry run).

Supports scanning for multiple media types:
- Images: jpg, png, gif, svg, webp, ico, bmp, tiff (enabled by default)
- Documents: pdf, doc, docx, ppt, pptx, xls, xlsx, txt, rtf, odt, ods, odp
- Videos: mp4, avi, mov, wmv, flv, webm, mkv, m4v, 3gp, ogv
- Audio: mp3, wav, ogg, flac, aac, m4a, wma, opus

Use --media-types to specify which types to scan, or use individual flags like --documents --videos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration merged with command flags
		cfg, err := mergeConfigWithFlags(cmd)
		if err != nil {
			return err
		}

		// Override source directory if provided as positional argument
		if len(args) > 0 {
			cfg.SourceDir = args[0]
		}

		// Get scan-specific flags
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		fmt.Printf("Scanning directory: %s\n", cfg.SourceDir)
		fmt.Printf("File extensions: %v\n", cfg.FileExtensions)

		if cfg.Verbose {
			fmt.Printf("Enabled media types: %v\n", cfg.EnabledMediaTypesStr)
		}

		// Create file scanner and scan for files
		fileScanner := scanner.NewFileScanner(cfg)
		files, err := fileScanner.ScanDirectory()
		if err != nil {
			return fmt.Errorf("error scanning directory: %w", err)
		}

		fmt.Printf("\nFound %d files to scan:\n", len(files))

		// Create link parser and process files using media-aware parsing
		linkParser := scanner.NewLinkParser()
		totalMedia := 0
		totalImages := 0 // For backward compatibility
		var allMediaReferences []types.MediaReference
		var allReferences []types.ImageReference // For backward compatibility

		for _, file := range files {
			// Use media-aware parsing
			mediaReferences, err := linkParser.ParseFileMedia(file, cfg)
			if err != nil {
				fmt.Printf("✗ %s (error: %v)\n", file, err)
				continue
			}

			// Count by media type for better reporting
			imageCount := 0
			documentCount := 0
			videoCount := 0
			audioCount := 0
			for _, ref := range mediaReferences {
				switch ref.MediaType {
				case types.MediaTypeImage:
					imageCount++
				case types.MediaTypeDocument:
					documentCount++
				case types.MediaTypeVideo:
					videoCount++
				case types.MediaTypeAudio:
					audioCount++
				}
			}

			if len(mediaReferences) > 0 {
				// Determine what to display based on enabled media types
				if len(cfg.EnabledMediaTypes) == 1 {
					// Single media type mode - show specific type
					mediaType := cfg.EnabledMediaTypes[0]
					var count int
					var typeName string
					switch mediaType {
					case types.MediaTypeImage:
						count = imageCount
						typeName = "images"
					case types.MediaTypeDocument:
						count = documentCount
						typeName = "documents"
					case types.MediaTypeVideo:
						count = videoCount
						typeName = "videos"
					case types.MediaTypeAudio:
						count = audioCount
						typeName = "audio files"
					}
					fmt.Printf("✓ %s (%d %s)\n", file, count, typeName)
				} else {
					// Multiple media types - show total
					fmt.Printf("✓ %s (%d media files)\n", file, len(mediaReferences))
				}

				if cfg.Verbose {
					for _, ref := range mediaReferences {
						fmt.Printf("  - Line %d: %s (%s)\n", ref.LineNumber, ref.OriginalURL, ref.MediaType.String())
					}
				}
			} else {
				// No media found
				if len(cfg.EnabledMediaTypes) == 1 {
					mediaType := cfg.EnabledMediaTypes[0]
					var typeName string
					switch mediaType {
					case types.MediaTypeImage:
						typeName = "images"
					case types.MediaTypeDocument:
						typeName = "documents"
					case types.MediaTypeVideo:
						typeName = "videos"
					case types.MediaTypeAudio:
						typeName = "audio files"
					}
					fmt.Printf("✓ %s (0 %s)\n", file, typeName)
				} else {
					fmt.Printf("✓ %s (0 media files)\n", file)
				}
			}

			allMediaReferences = append(allMediaReferences, mediaReferences...)
			totalMedia += len(mediaReferences)
			totalImages += imageCount
		}

		fmt.Printf("\nSummary:\n")
		fmt.Printf("- Files scanned: %d\n", len(files))

		// Calculate stats by media type
		mediaStats := make(map[types.MediaType]int)
		for _, ref := range allMediaReferences {
			mediaStats[ref.MediaType]++
		}

		if len(cfg.EnabledMediaTypes) == 1 {
			// Single media type mode - show specific type total
			mediaType := cfg.EnabledMediaTypes[0]
			count := mediaStats[mediaType]
			var typeName string
			switch mediaType {
			case types.MediaTypeImage:
				typeName = "Images"
			case types.MediaTypeDocument:
				typeName = "Documents"
			case types.MediaTypeVideo:
				typeName = "Videos"
			case types.MediaTypeAudio:
				typeName = "Audio files"
			}
			fmt.Printf("- %s found: %d\n", typeName, count)
		} else {
			// Multiple media types - show total and breakdown
			fmt.Printf("- Media files found: %d\n", totalMedia)
			for mediaType, count := range mediaStats {
				if count > 0 {
					fmt.Printf("  - %s: %d\n", mediaType.String(), count)
				}
			}
		}

		// Save results to file if requested
		if outputFile != "" {
			// Convert MediaReference to ImageReference for backward compatibility in save format
			for _, mediaRef := range allMediaReferences {
				if mediaRef.MediaType == types.MediaTypeImage {
					imageRef := types.ImageReference{
						SourceFile:   mediaRef.SourceFile,
						LocalPath:    mediaRef.LocalPath,
						LineNumber:   mediaRef.LineNumber,
						OriginalURL:  mediaRef.OriginalURL,
						NewURL:       mediaRef.NewURL,
						FileSize:     mediaRef.FileSize,
						ContentType:  mediaRef.ContentType,
						LastModified: mediaRef.LastModified,
					}
					allReferences = append(allReferences, imageRef)
				}
			}

			results := &ScanResults{
				Timestamp:       time.Now(),
				SourceDir:       cfg.SourceDir,
				Extensions:      cfg.FileExtensions,
				TotalFiles:      len(files),
				TotalMedia:      totalMedia,
				TotalImages:     totalImages,
				MediaReferences: allMediaReferences,
				References:      allReferences, // Backward compatibility
				MediaStats:      make(map[string]int),
			}

			// Populate media stats
			for _, ref := range allMediaReferences {
				results.MediaStats[ref.MediaType.String()]++
			}

			if err := saveScanResults(results, outputFile, format); err != nil {
				return fmt.Errorf("error saving scan results: %w", err)
			}

			fmt.Printf("- Scan results saved to: %s (%s format)\n", outputFile, format)
		}

		return nil
	},
} // verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify S3 bucket configuration",
	Long:  `Verify S3 bucket configuration and test access permissions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration merged with command flags
		cfg, err := mergeConfigWithFlags(cmd)
		if err != nil {
			return err
		}

		// Validate required configuration
		if cfg.S3Bucket == "" {
			return fmt.Errorf("S3 bucket is required (use --bucket flag or set s3_bucket in config)")
		}

		fmt.Printf("Verifying S3 configuration...\n")
		fmt.Printf("Bucket: %s\n", cfg.S3Bucket)
		fmt.Printf("Region: %s\n", cfg.S3Region)
		fmt.Println()

		// Create S3 configuration
		config := &types.MigrationConfig{
			S3Bucket: cfg.S3Bucket,
			S3Region: cfg.S3Region,
			Verbose:  cfg.Verbose,
		}

		// Create S3 client
		s3Client, err := storage.NewS3Client(config)
		if err != nil {
			return fmt.Errorf("failed to create S3 client: %w", err)
		}

		// Test 1: Validate AWS credentials and configuration
		fmt.Printf("✓ Testing AWS credentials and configuration...")
		if err := s3Client.ValidateConfiguration(); err != nil {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("AWS credentials validation failed: %w", err)
		}
		fmt.Printf(" ✓ SUCCESS\n")

		// Test 2: Check if bucket exists
		fmt.Printf("✓ Checking if bucket '%s' exists...", cfg.S3Bucket)
		exists, err := s3Client.BucketExists(cfg.S3Bucket)
		if err != nil {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("error checking bucket existence: %w", err)
		}

		if !exists {
			fmt.Printf(" ✗ BUCKET NOT FOUND\n")
			fmt.Printf("\nBucket '%s' does not exist.\n", cfg.S3Bucket)
			fmt.Printf("You can create it manually or using: aws s3 mb s3://%s --region %s\n", cfg.S3Bucket, cfg.S3Region)
			return fmt.Errorf("bucket does not exist")
		}
		fmt.Printf(" ✓ SUCCESS\n")

		// Test 3: Test upload permissions (create a small test file)
		fmt.Printf("✓ Testing upload permissions...")
		testKey := "mv2s3-test-" + fmt.Sprintf("%d", time.Now().Unix()) + ".txt"
		testContent := "This is a test file created by mv2s3 to verify upload permissions."

		// Create temporary test file
		tmpFile, err := os.CreateTemp("", "mv2s3-test-*.txt")
		if err != nil {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("failed to create test file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		if _, err := tmpFile.WriteString(testContent); err != nil {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("failed to write test file: %w", err)
		}
		tmpFile.Close()

		// Try to upload test file
		_, err = s3Client.UploadFile(tmpFile.Name(), cfg.S3Bucket, testKey)
		if err != nil {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("upload test failed: %w", err)
		}
		fmt.Printf(" ✓ SUCCESS\n")

		// Test 4: Test if object exists
		fmt.Printf("✓ Testing object existence check...")
		exists, err = s3Client.ObjectExists(cfg.S3Bucket, testKey)
		if err != nil {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("object existence check failed: %w", err)
		}
		if !exists {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("uploaded object not found")
		}
		fmt.Printf(" ✓ SUCCESS\n")

		// Test 5: Test public URL generation
		fmt.Printf("✓ Testing URL generation...")
		publicURL := s3Client.GetPublicURL(cfg.S3Bucket, testKey)
		if publicURL == "" {
			fmt.Printf(" ✗ FAILED\n")
			return fmt.Errorf("failed to generate public URL")
		}
		fmt.Printf(" ✓ SUCCESS\n")
		fmt.Printf("  Sample URL: %s\n", publicURL)

		// Test 6: Cleanup test file
		fmt.Printf("✓ Cleaning up test file...")
		if err := s3Client.DeleteObject(cfg.S3Bucket, testKey); err != nil {
			fmt.Printf(" ✗ FAILED\n")
			fmt.Printf("  Warning: Failed to delete test file '%s': %v\n", testKey, err)
		} else {
			fmt.Printf(" ✓ SUCCESS\n")
		}

		fmt.Printf("\n🎉 All S3 configuration tests passed!\n")
		fmt.Printf("✓ AWS credentials are valid\n")
		fmt.Printf("✓ Bucket '%s' exists and is accessible\n", cfg.S3Bucket)
		fmt.Printf("✓ Upload permissions are working\n")
		fmt.Printf("✓ Object operations are working\n")
		fmt.Printf("✓ URL generation is working\n")
		fmt.Printf("\nYour S3 configuration is ready for migration!\n")

		return nil
	},
}

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up local image files",
	Long:  `Clean up local image files after successful migration to S3.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration merged with command flags
		cfg, err := mergeConfigWithFlags(cmd)
		if err != nil {
			return err
		}

		// Get cleanup-specific flags
		scanFile, _ := cmd.Flags().GetString("scan-file")
		force, _ := cmd.Flags().GetBool("force")
		verifyS3, _ := cmd.Flags().GetBool("verify-s3")

		fmt.Printf("Starting cleanup process...\n")
		if cfg.DryRun {
			fmt.Printf("DRY RUN MODE - No files will be deleted\n")
		}
		fmt.Println()

		// Load scan results
		if scanFile == "" {
			return fmt.Errorf("--scan-file is required for cleanup operation")
		}

		fmt.Printf("Loading scan results from: %s\n", scanFile)
		results, err := loadScanResults(scanFile)
		if err != nil {
			return fmt.Errorf("error loading scan file: %w", err)
		}

		fmt.Printf("Loaded %d image references from scan file\n", len(results.References))

		if len(results.References) == 0 {
			fmt.Println("No image references found in scan file.")
			return nil
		}

		// Initialize S3 client if verification is requested
		var s3Client storage.StorageClient
		if verifyS3 {
			if cfg.S3Bucket == "" {
				return fmt.Errorf("S3 bucket is required when --verify-s3 is enabled (use --bucket flag or set s3_bucket in config)")
			}

			fmt.Printf("Initializing S3 client for verification...\n")
			migrationConfig := &types.MigrationConfig{
				S3Bucket: cfg.S3Bucket,
				S3Region: cfg.S3Region,
				S3Prefix: cfg.S3Prefix,
			}
			s3Client, err = storage.NewS3Client(migrationConfig)
			if err != nil {
				return fmt.Errorf("error creating S3 client: %w", err)
			}
		}

		// Process each image reference
		fmt.Printf("\nAnalyzing images for cleanup...\n")

		var toDelete []string
		var notFound []string
		var s3Missing []string

		for _, ref := range results.References {
			// Construct full path to local image
			var imagePath string
			if filepath.IsAbs(ref.OriginalURL) {
				imagePath = ref.OriginalURL
			} else {
				// Resolve relative to source file's directory
				sourceDir := filepath.Dir(ref.SourceFile)
				imagePath = filepath.Join(sourceDir, ref.OriginalURL)
			}

			// Check if local file exists
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				notFound = append(notFound, imagePath)
				if verbose {
					fmt.Printf("  ⚠ Local file not found: %s\n", imagePath)
				}
				continue
			}

			// Verify S3 existence if requested
			if verifyS3 {
				s3Key := cleanS3Key(ref.OriginalURL, cfg.S3Prefix)
				exists, err := s3Client.ObjectExists(cfg.S3Bucket, s3Key)
				if err != nil {
					fmt.Printf("  ✗ Error checking S3 object %s: %v\n", s3Key, err)
					continue
				}
				if !exists {
					s3Missing = append(s3Missing, imagePath)
					if cfg.Verbose {
						fmt.Printf("  ⚠ S3 object not found: s3://%s/%s (local: %s)\n", cfg.S3Bucket, s3Key, imagePath)
					}
					continue
				}
			}

			toDelete = append(toDelete, imagePath)
			if cfg.Verbose {
				fmt.Printf("  ✓ Marked for deletion: %s\n", imagePath)
			}
		}

		// Summary
		fmt.Printf("\nCleanup Analysis:\n")
		fmt.Printf("- Total images in scan: %d\n", len(results.References))
		fmt.Printf("- Local files found: %d\n", len(results.References)-len(notFound))
		fmt.Printf("- Local files not found: %d\n", len(notFound))
		if verifyS3 {
			fmt.Printf("- S3 verification enabled: ✓\n")
			fmt.Printf("- Files missing from S3: %d\n", len(s3Missing))
		} else {
			fmt.Printf("- S3 verification enabled: ✗ (use --verify-s3 to enable)\n")
		}
		fmt.Printf("- Files to delete: %d\n", len(toDelete))

		if len(toDelete) == 0 {
			fmt.Println("\nNo files to clean up.")
			return nil
		}

		// Show warnings if any
		if len(s3Missing) > 0 && !force {
			fmt.Printf("\n⚠ WARNING: %d files are missing from S3 and will NOT be deleted:\n", len(s3Missing))
			for _, file := range s3Missing {
				fmt.Printf("  - %s\n", file)
			}
			fmt.Printf("\nUse --force to delete local files even if S3 verification fails.\n")
		}

		if len(notFound) > 0 && cfg.Verbose {
			fmt.Printf("\nLocal files not found (already deleted?):\n")
			for _, file := range notFound {
				fmt.Printf("  - %s\n", file)
			}
		}

		if cfg.DryRun {
			fmt.Printf("\n[DRY RUN] Would delete %d files:\n", len(toDelete))
			for _, file := range toDelete {
				fmt.Printf("  - %s\n", file)
			}
			fmt.Println("\n✓ Dry run completed - no files deleted")
			return nil
		}

		// Confirm deletion unless force is used
		if !force {
			fmt.Printf("\n⚠ This will permanently delete %d local image files.\n", len(toDelete))
			fmt.Printf("Continue? (y/N): ")

			var response string
			fmt.Scanln(&response)
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Println("Cleanup cancelled.")
				return nil
			}
		}

		// Create backups if requested
		if cfg.BackupOriginals {
			fmt.Printf("\nCreating backups in: %s\n", cfg.BackupDir)
			if err := os.MkdirAll(cfg.BackupDir, 0755); err != nil {
				return fmt.Errorf("failed to create backup directory: %w", err)
			}
		}

		// Delete files
		fmt.Printf("\nDeleting local image files...\n")
		deletedCount := 0
		failedCount := 0

		for i, imagePath := range toDelete {
			fmt.Printf("  [%d/%d] Deleting %s", i+1, len(toDelete), imagePath)

			// Create backup if requested
			if cfg.BackupOriginals {
				relPath, err := filepath.Rel(results.SourceDir, imagePath)
				if err != nil {
					relPath = filepath.Base(imagePath)
				}
				backupPath := filepath.Join(cfg.BackupDir, relPath)

				// Ensure backup directory exists
				if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
					fmt.Printf(" ✗ (backup failed: %v)\n", err)
					failedCount++
					continue
				}

				// Copy to backup
				if err := copyFile(imagePath, backupPath); err != nil {
					fmt.Printf(" ✗ (backup failed: %v)\n", err)
					failedCount++
					continue
				}
			}

			// Delete original file
			if err := os.Remove(imagePath); err != nil {
				fmt.Printf(" ✗ (delete failed: %v)\n", err)
				failedCount++
				continue
			}

			fmt.Printf(" ✓\n")
			deletedCount++
		}

		// Final summary
		fmt.Printf("\n✓ Cleanup completed!\n")
		fmt.Printf("Summary:\n")
		fmt.Printf("- Files successfully deleted: %d\n", deletedCount)
		fmt.Printf("- Failed deletions: %d\n", failedCount)
		if cfg.BackupOriginals {
			fmt.Printf("- Backup location: %s\n", cfg.BackupDir)
		}

		return nil
	},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `View and manage mv2s3 configuration.`,
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new configuration file with default values or from template.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive, _ := cmd.Flags().GetBool("interactive")
		template, _ := cmd.Flags().GetString("template")
		configPath, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")

		// Determine config file path
		if configPath == "" {
			if cfgFile != "" {
				configPath = cfgFile
			} else {
				configPath = ".mv2s3.yaml"
			}
		}

		// Check if file already exists
		if _, err := os.Stat(configPath); err == nil && !force {
			return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
		}

		fmt.Printf("Initializing configuration file: %s\n", configPath)

		// Create config with defaults
		var configData *types.MigrationConfig
		if template != "" {
			fmt.Printf("Using template: %s\n", template)
			configData = getConfigTemplate(template)
			if configData == nil {
				return fmt.Errorf("unknown template: %s", template)
			}
		} else {
			configData = getDefaultConfig()
		}

		// Interactive mode
		if interactive {
			fmt.Println("\nInteractive configuration setup:")
			configData = runInteractiveSetup(configData)
		}

		// Save config file
		if err := saveConfigFile(configData, configPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("\n✓ Configuration file created successfully!\n")
		fmt.Printf("Location: %s\n", configPath)
		fmt.Printf("\nYou can now:\n")
		fmt.Printf("- Edit the file manually\n")
		fmt.Printf("- Use 'mv2s3 config show' to view current settings\n")
		fmt.Printf("- Use 'mv2s3 config set <key> <value>' to modify settings\n")

		return nil
	},
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show [key]",
	Short: "Show current configuration",
	Long:  `Display current configuration values from all sources (file, environment, defaults).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		sources, _ := cmd.Flags().GetBool("sources")

		// Load configuration
		cfg, configFileUsed, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Show specific key if provided
		if len(args) > 0 {
			key := args[0]
			return showConfigKey(cfg, key, format, sources, configFileUsed)
		}

		// Show all configuration
		return showAllConfig(cfg, format, sources, configFileUsed)
	},
}

// configGetCmd represents the config get command
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a specific configuration value",
	Long:  `Get a specific configuration value by key name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		sources, _ := cmd.Flags().GetBool("sources")

		// Load configuration
		cfg, configFileUsed, err := config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		key := args[0]
		return showConfigKey(cfg, key, format, sources, configFileUsed)
	},
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  `Set a configuration value by key name. The configuration file will be updated.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		global, _ := cmd.Flags().GetBool("global")

		// Determine config file path
		configPath := cfgFile
		if configPath == "" {
			if global {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}
				configPath = filepath.Join(homeDir, ".mv2s3.yaml")
			} else {
				configPath = ".mv2s3.yaml"
			}
		}

		// Load existing configuration or create new one
		var cfg *types.MigrationConfig
		if _, err := os.Stat(configPath); err == nil {
			cfg, _, err = config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load existing configuration: %w", err)
			}
		} else {
			cfg = getDefaultConfig()
		}

		// Set the value
		if err := setConfigValue(cfg, key, value); err != nil {
			return fmt.Errorf("failed to set configuration value: %w", err)
		}

		// Save the configuration
		if err := saveConfigFile(cfg, configPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("✓ Configuration updated: %s = %s\n", key, value)
		fmt.Printf("File: %s\n", configPath)

		return nil
	},
}

// configMigrateCmd represents the config migrate command
var configMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate v0.1.x configuration to v0.2.0 format",
	Long: `Migrate an existing v0.1.x configuration file to the new v0.2.0 format with media type support.

This command:
- Detects v0.1.x configuration files
- Converts them to v0.2.0 format with media type configuration
- Preserves all existing settings
- Enables backward compatibility
- Creates a backup of the original configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the config file path
		configPath := cfgFile
		if configPath == "" {
			if len(args) > 0 {
				configPath = args[0]
			} else {
				configPath = ".mv2s3.yaml"
			}
		}

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("configuration file not found: %s", configPath)
		}

		// Load the existing configuration
		cfg, _, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if already v0.2.0 format
		if len(cfg.MediaTypes) > 0 {
			fmt.Printf("✓ Configuration is already in v0.2.0 format: %s\n", configPath)
			return nil
		}

		// Create backup
		backupPath := configPath + ".v1.backup"
		if err := copyFile(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("✓ Created backup: %s\n", backupPath)

		// The configuration has already been migrated by the LoadConfig function
		// We just need to save it in the new format
		if err := saveConfigFile(cfg, configPath); err != nil {
			return fmt.Errorf("failed to save migrated configuration: %w", err)
		}

		fmt.Printf("✓ Configuration migrated to v0.2.0 format: %s\n", configPath)
		fmt.Printf("\nMigration summary:\n")
		fmt.Printf("- Enabled media types: %v\n", cfg.EnabledMediaTypesStr)
		fmt.Printf("- Backward compatibility: Preserved all v0.1.x settings\n")
		fmt.Printf("- Images are enabled by default (matches v0.1.x behavior)\n")
		fmt.Printf("- Other media types are disabled by default\n")
		fmt.Printf("\nTo enable additional media types, edit the configuration file or use:\n")
		fmt.Printf("  mv2s3 config set enabled_media_types 'images,documents,videos'\n")

		return nil
	},
}

// getDefaultConfig returns a configuration with default values
func getDefaultConfig() *types.MigrationConfig {
	return &types.MigrationConfig{
		SourceDir:       ".",
		FileExtensions:  []string{"html", "css", "js", "md", "jsx", "tsx"},
		ExcludePatterns: []string{"node_modules/**", ".git/**", "dist/**", "build/**"},
		IncludePatterns: []string{},

		S3Region:  "us-east-1",
		S3Prefix:  "images/",
		S3Profile: "default",

		DryRun:          false,
		CleanupLocal:    false,
		BackupOriginals: true,
		BackupDir:       "./backups",
		Concurrency:     5,

		LogLevel:  "info",
		LogFormat: "text",
		Verbose:   false,
	}
}

// getConfigTemplate returns a template configuration
func getConfigTemplate(template string) *types.MigrationConfig {
	switch strings.ToLower(template) {
	case "hugo":
		return &types.MigrationConfig{
			SourceDir:       ".",
			FileExtensions:  []string{"md", "html", "css", "js"},
			ExcludePatterns: []string{"public/**", "resources/**", ".git/**"},
			IncludePatterns: []string{},

			S3Region:  "us-east-1",
			S3Prefix:  "images/",
			S3Profile: "default",

			DryRun:          false,
			CleanupLocal:    false,
			BackupOriginals: true,
			BackupDir:       "./backups",
			Concurrency:     3,

			LogLevel:  "info",
			LogFormat: "text",
			Verbose:   false,
		}
	case "nextjs", "next":
		return &types.MigrationConfig{
			SourceDir:       ".",
			FileExtensions:  []string{"js", "jsx", "ts", "tsx", "md", "mdx"},
			ExcludePatterns: []string{".next/**", "node_modules/**", ".git/**"},
			IncludePatterns: []string{},

			S3Region:  "us-east-1",
			S3Prefix:  "static/",
			S3Profile: "default",

			DryRun:          false,
			CleanupLocal:    false,
			BackupOriginals: true,
			BackupDir:       "./backups",
			Concurrency:     5,

			LogLevel:  "info",
			LogFormat: "text",
			Verbose:   false,
		}
	case "gatsby":
		return &types.MigrationConfig{
			SourceDir:       "src",
			FileExtensions:  []string{"js", "jsx", "ts", "tsx", "md", "mdx"},
			ExcludePatterns: []string{"public/**", ".cache/**", "node_modules/**", ".git/**"},
			IncludePatterns: []string{},

			S3Region:  "us-east-1",
			S3Prefix:  "images/",
			S3Profile: "default",

			DryRun:          false,
			CleanupLocal:    false,
			BackupOriginals: true,
			BackupDir:       "./backups",
			Concurrency:     5,

			LogLevel:  "info",
			LogFormat: "text",
			Verbose:   false,
		}
	default:
		return nil
	}
}

// runInteractiveSetup prompts user for configuration values
func runInteractiveSetup(cfg *types.MigrationConfig) *types.MigrationConfig {
	var input string

	// S3 Bucket
	fmt.Printf("S3 Bucket name (required): ")
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		cfg.S3Bucket = input
	}

	// S3 Region
	fmt.Printf("S3 Region [%s]: ", cfg.S3Region)
	input = ""
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		cfg.S3Region = input
	}

	// S3 Prefix
	fmt.Printf("S3 Key prefix [%s]: ", cfg.S3Prefix)
	input = ""
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		cfg.S3Prefix = input
	}

	// Source Directory
	fmt.Printf("Source directory [%s]: ", cfg.SourceDir)
	input = ""
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		cfg.SourceDir = input
	}

	// File Extensions
	fmt.Printf("File extensions [%s]: ", strings.Join(cfg.FileExtensions, ","))
	input = ""
	if _, err := fmt.Scanln(&input); err == nil && input != "" {
		cfg.FileExtensions = strings.Split(input, ",")
		// Trim spaces
		for i, ext := range cfg.FileExtensions {
			cfg.FileExtensions[i] = strings.TrimSpace(ext)
		}
	}

	return cfg
} // saveConfigFile saves configuration to YAML file
func saveConfigFile(cfg *types.MigrationConfig, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// showConfigKey displays a specific configuration key
func showConfigKey(cfg *types.MigrationConfig, key, format string, showSources bool, configFileUsed string) error {
	value, found := getConfigValue(cfg, key)
	if !found {
		return fmt.Errorf("configuration key '%s' not found", key)
	}

	switch format {
	case "json":
		data := map[string]interface{}{key: value}
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	case "yaml":
		data := map[string]interface{}{key: value}
		yamlData, err := yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(yamlData))
	default:
		fmt.Printf("%s: %v\n", key, value)
	}

	if showSources {
		fmt.Println("\nConfiguration Sources:")
		if configFileUsed != "" {
			fmt.Printf("  config_file: %s\n", configFileUsed)
		} else {
			homeDir, _ := os.UserHomeDir()
			defaultConfigPath := filepath.Join(homeDir, ".mv2s3.yaml")
			fmt.Printf("  config_file: %s (default location)\n", defaultConfigPath)
		}
		fmt.Printf("  environment: MV2S3_* variables\n")
	}

	return nil
}

// showAllConfig displays all configuration values
func showAllConfig(cfg *types.MigrationConfig, format string, showSources bool, configFileUsed string) error {
	switch format {
	case "json":
		jsonData, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	case "yaml":
		yamlData, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Print(string(yamlData))
	default:
		// Human-readable format
		fmt.Println("Current Configuration:")
		fmt.Println("======================")

		fmt.Println("\nSource Scanning:")
		fmt.Printf("  source_dir: %s\n", cfg.SourceDir)
		fmt.Printf("  file_extensions: %v\n", cfg.FileExtensions)
		if len(cfg.ExcludePatterns) > 0 {
			fmt.Printf("  exclude_patterns: %v\n", cfg.ExcludePatterns)
		}
		if len(cfg.IncludePatterns) > 0 {
			fmt.Printf("  include_patterns: %v\n", cfg.IncludePatterns)
		}

		fmt.Println("\nAWS S3 Configuration:")
		if cfg.S3Bucket != "" {
			fmt.Printf("  s3_bucket: %s\n", cfg.S3Bucket)
		} else {
			fmt.Printf("  s3_bucket: (not set)\n")
		}
		fmt.Printf("  s3_region: %s\n", cfg.S3Region)
		if cfg.S3Prefix != "" {
			fmt.Printf("  s3_prefix: %s\n", cfg.S3Prefix)
		}
		if cfg.S3Profile != "" {
			fmt.Printf("  s3_profile: %s\n", cfg.S3Profile)
		}
		if cfg.S3Endpoint != "" {
			fmt.Printf("  s3_endpoint: %s\n", cfg.S3Endpoint)
		}

		fmt.Println("\nProcessing Options:")
		fmt.Printf("  dry_run: %t\n", cfg.DryRun)
		fmt.Printf("  cleanup_local: %t\n", cfg.CleanupLocal)
		fmt.Printf("  backup_originals: %t\n", cfg.BackupOriginals)
		fmt.Printf("  backup_dir: %s\n", cfg.BackupDir)
		fmt.Printf("  concurrency: %d\n", cfg.Concurrency)

		fmt.Println("\nLogging:")
		fmt.Printf("  log_level: %s\n", cfg.LogLevel)
		fmt.Printf("  log_format: %s\n", cfg.LogFormat)
		fmt.Printf("  verbose: %t\n", cfg.Verbose)

		if showSources {
			fmt.Println("\nConfiguration Sources:")
			if configFileUsed != "" {
				fmt.Printf("  config_file: %s\n", configFileUsed)
			} else {
				homeDir, _ := os.UserHomeDir()
				defaultConfigPath := filepath.Join(homeDir, ".mv2s3.yaml")
				fmt.Printf("  config_file: %s (default location)\n", defaultConfigPath)
			}
			fmt.Printf("  environment: MV2S3_* variables\n")
		}
	}

	return nil
}

// getConfigValue gets a configuration value by key name
func getConfigValue(cfg *types.MigrationConfig, key string) (interface{}, bool) {
	switch strings.ToLower(key) {
	case "source_dir", "sourcedir":
		return cfg.SourceDir, true
	case "file_extensions", "fileextensions":
		return cfg.FileExtensions, true
	case "exclude_patterns", "excludepatterns":
		return cfg.ExcludePatterns, true
	case "include_patterns", "includepatterns":
		return cfg.IncludePatterns, true
	case "s3_bucket", "s3bucket":
		return cfg.S3Bucket, true
	case "s3_region", "s3region":
		return cfg.S3Region, true
	case "s3_prefix", "s3prefix":
		return cfg.S3Prefix, true
	case "s3_endpoint", "s3endpoint":
		return cfg.S3Endpoint, true
	case "s3_access_key", "s3accesskey":
		return cfg.S3AccessKey, true
	case "s3_secret_key", "s3secretkey":
		return "***hidden***", true
	case "s3_profile", "s3profile":
		return cfg.S3Profile, true
	case "dry_run", "dryrun":
		return cfg.DryRun, true
	case "cleanup_local", "cleanuplocal":
		return cfg.CleanupLocal, true
	case "backup_originals", "backuporiginals":
		return cfg.BackupOriginals, true
	case "backup_dir", "backupdir":
		return cfg.BackupDir, true
	case "concurrency":
		return cfg.Concurrency, true
	case "log_level", "loglevel":
		return cfg.LogLevel, true
	case "log_format", "logformat":
		return cfg.LogFormat, true
	case "verbose":
		return cfg.Verbose, true
	default:
		return nil, false
	}
}

// setConfigValue sets a configuration value by key name
func setConfigValue(cfg *types.MigrationConfig, key, value string) error {
	switch strings.ToLower(key) {
	case "source_dir", "sourcedir":
		cfg.SourceDir = value
	case "file_extensions", "fileextensions":
		cfg.FileExtensions = strings.Split(value, ",")
		// Trim spaces
		for i, ext := range cfg.FileExtensions {
			cfg.FileExtensions[i] = strings.TrimSpace(ext)
		}
	case "exclude_patterns", "excludepatterns":
		cfg.ExcludePatterns = strings.Split(value, ",")
		// Trim spaces
		for i, pattern := range cfg.ExcludePatterns {
			cfg.ExcludePatterns[i] = strings.TrimSpace(pattern)
		}
	case "include_patterns", "includepatterns":
		cfg.IncludePatterns = strings.Split(value, ",")
		// Trim spaces
		for i, pattern := range cfg.IncludePatterns {
			cfg.IncludePatterns[i] = strings.TrimSpace(pattern)
		}
	case "s3_bucket", "s3bucket":
		cfg.S3Bucket = value
	case "s3_region", "s3region":
		cfg.S3Region = value
	case "s3_prefix", "s3prefix":
		cfg.S3Prefix = value
	case "s3_endpoint", "s3endpoint":
		cfg.S3Endpoint = value
	case "s3_access_key", "s3accesskey":
		cfg.S3AccessKey = value
	case "s3_secret_key", "s3secretkey":
		cfg.S3SecretKey = value
	case "s3_profile", "s3profile":
		cfg.S3Profile = value
	case "dry_run", "dryrun":
		val, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.DryRun = val
	case "cleanup_local", "cleanuplocal":
		val, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.CleanupLocal = val
	case "backup_originals", "backuporiginals":
		val, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.BackupOriginals = val
	case "backup_dir", "backupdir":
		cfg.BackupDir = value
	case "concurrency":
		val, err := parsePositiveInt(value)
		if err != nil {
			return fmt.Errorf("invalid concurrency value: %s", value)
		}
		cfg.Concurrency = val
	case "log_level", "loglevel":
		if !isValidLogLevel(value) {
			return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", value)
		}
		cfg.LogLevel = value
	case "log_format", "logformat":
		if !isValidLogFormat(value) {
			return fmt.Errorf("invalid log format: %s (valid: text, json)", value)
		}
		cfg.LogFormat = value
	case "verbose":
		val, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.Verbose = val
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}

// parseBool parses a boolean value from string
func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "t", "yes", "y", "1", "on", "enable", "enabled":
		return true, nil
	case "false", "f", "no", "n", "0", "off", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", value)
	}
}

// parsePositiveInt parses a positive integer from string
func parsePositiveInt(value string) (int, error) {
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if val <= 0 {
		return 0, fmt.Errorf("value must be positive: %d", val)
	}
	return val, nil
}

// isValidLogLevel checks if log level is valid
func isValidLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

// isValidLogFormat checks if log format is valid
func isValidLogFormat(format string) bool {
	switch strings.ToLower(format) {
	case "text", "json":
		return true
	default:
		return false
	}
}

func init() {
	// Add flags to migrate command
	migrateCmd.Flags().StringP("source", "s", ".", "Source directory to scan")
	migrateCmd.Flags().StringP("bucket", "b", "", "S3 bucket name (required)")
	migrateCmd.Flags().StringP("region", "r", "us-east-1", "AWS region")
	migrateCmd.Flags().StringP("prefix", "p", "", "S3 key prefix")
	migrateCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	migrateCmd.Flags().Bool("cleanup", false, "Remove local files after successful upload")
	migrateCmd.Flags().Bool("backup", true, "Create backups of original files")
	migrateCmd.Flags().StringSlice("extensions", []string{"html", "css", "js", "md"}, "File extensions to scan")
	migrateCmd.Flags().String("scan-file", "", "Use pre-generated scan file instead of scanning (optional)")

	// Media type filtering flags (v0.2.0+)
	migrateCmd.Flags().Bool("images", true, "Migrate image files (jpg, png, gif, svg, etc.)")
	migrateCmd.Flags().Bool("documents", false, "Migrate document files (pdf, doc, docx, etc.)")
	migrateCmd.Flags().Bool("videos", false, "Migrate video files (mp4, avi, mov, etc.)")
	migrateCmd.Flags().Bool("audio", false, "Migrate audio files (mp3, wav, ogg, etc.)")
	migrateCmd.Flags().StringSlice("media-types", []string{"images"}, "Media types to migrate: images, documents, videos, audio")
	migrateCmd.Flags().StringSlice("include-extensions", []string{}, "Additional file extensions to include for migration")

	// Add flags to scan command
	scanCmd.Flags().StringP("source", "s", ".", "Source directory to scan")
	scanCmd.Flags().StringSlice("extensions", []string{"md", "markdown", "html", "css"}, "File extensions to scan (prioritized for Hugo sites)")
	scanCmd.Flags().StringP("output", "o", "", "Save scan results to file (optional)")
	scanCmd.Flags().String("format", "text", "Output format: text, json, csv")

	// Media type filtering flags for scan command (v0.2.0+)
	scanCmd.Flags().Bool("images", true, "Scan for image references (jpg, png, gif, svg, etc.)")
	scanCmd.Flags().Bool("documents", false, "Scan for document references (pdf, doc, docx, etc.)")
	scanCmd.Flags().Bool("videos", false, "Scan for video references (mp4, avi, mov, etc.)")
	scanCmd.Flags().Bool("audio", false, "Scan for audio references (mp3, wav, ogg, etc.)")
	scanCmd.Flags().StringSlice("media-types", []string{"images"}, "Media types to scan: images, documents, videos, audio")

	// Add flags to verify command
	verifyCmd.Flags().StringP("bucket", "b", "", "S3 bucket name (required)")
	verifyCmd.Flags().StringP("region", "r", "us-east-1", "AWS region")

	// Add flags to cleanup command
	cleanupCmd.Flags().String("scan-file", "", "Scan file containing migrated images (required)")
	cleanupCmd.Flags().StringP("bucket", "b", "", "S3 bucket name (required for S3 verification)")
	cleanupCmd.Flags().StringP("region", "r", "us-east-1", "AWS region")
	cleanupCmd.Flags().StringP("prefix", "p", "", "S3 key prefix")
	cleanupCmd.Flags().Bool("dry-run", false, "Show what would be deleted without making changes")
	cleanupCmd.Flags().Bool("backup", true, "Create backups before deleting files")
	cleanupCmd.Flags().String("backup-dir", "./backups", "Directory for backups")
	cleanupCmd.Flags().Bool("force", false, "Skip confirmation prompts")
	cleanupCmd.Flags().Bool("verify-s3", true, "Verify files exist on S3 before deleting locally")
	cleanupCmd.MarkFlagRequired("scan-file")

	// Add config subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configMigrateCmd)

	// Add flags to config init command
	configInitCmd.Flags().Bool("interactive", false, "Run interactive setup")
	configInitCmd.Flags().String("template", "", "Use template: hugo, nextjs, gatsby")
	configInitCmd.Flags().String("output", "mv2s3.yaml", "Output configuration file")
	configInitCmd.Flags().Bool("force", false, "Overwrite existing configuration file")

	// Add flags to config show command
	configShowCmd.Flags().String("format", "text", "Output format: text, json, yaml")
	configShowCmd.Flags().String("key", "", "Show specific configuration key")
	configShowCmd.Flags().Bool("sources", false, "Show configuration sources")

	// Add flags to config get command
	configGetCmd.Flags().String("format", "text", "Output format: text, json, yaml")
	configGetCmd.Flags().Bool("sources", false, "Show configuration sources")

	// Add flags to config set command
	configSetCmd.Flags().Bool("global", false, "Set in global configuration file (~/.mv2s3.yaml)")
}

// saveScanResults saves scan results to a file in the specified format
func saveScanResults(results *ScanResults, outputFile, format string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch strings.ToLower(format) {
	case "json":
		return saveScanResultsJSON(results, outputFile)
	case "csv":
		return saveScanResultsCSV(results, outputFile)
	case "text":
		return saveScanResultsText(results, outputFile)
	default:
		return fmt.Errorf("unsupported format: %s (supported: text, json, csv)", format)
	}
}

// saveScanResultsJSON saves results as JSON
func saveScanResultsJSON(results *ScanResults, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// saveScanResultsCSV saves results as CSV
func saveScanResultsCSV(results *ScanResults, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"source_file", "line_number", "original_url", "local_path"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, ref := range results.References {
		record := []string{
			ref.SourceFile,
			fmt.Sprintf("%d", ref.LineNumber),
			ref.OriginalURL,
			ref.LocalPath,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// saveScanResultsText saves results as human-readable text
func saveScanResultsText(results *ScanResults, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header information
	fmt.Fprintf(file, "# mv2s3 Scan Results\n")
	fmt.Fprintf(file, "# Generated: %s\n", results.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(file, "# Source Directory: %s\n", results.SourceDir)
	fmt.Fprintf(file, "# File Extensions: %s\n", strings.Join(results.Extensions, ", "))
	fmt.Fprintf(file, "# Total Files: %d\n", results.TotalFiles)
	fmt.Fprintf(file, "# Total Images: %d\n", results.TotalImages)
	fmt.Fprintf(file, "#\n")
	fmt.Fprintf(file, "# Format: source_file:line_number:original_url\n")
	fmt.Fprintf(file, "#\n")

	// Write image references
	for _, ref := range results.References {
		fmt.Fprintf(file, "%s:%d:%s\n", ref.SourceFile, ref.LineNumber, ref.OriginalURL)
	}

	return nil
}

// loadScanResults loads scan results from a file
func loadScanResults(scanFile string) (*ScanResults, error) {
	ext := strings.ToLower(filepath.Ext(scanFile))

	switch ext {
	case ".json":
		return loadScanResultsJSON(scanFile)
	case ".csv":
		return loadScanResultsCSV(scanFile)
	case ".txt":
		return loadScanResultsText(scanFile)
	default:
		// Try to detect format by content
		return loadScanResultsAuto(scanFile)
	}
}

// loadScanResultsJSON loads results from JSON file
func loadScanResultsJSON(scanFile string) (*ScanResults, error) {
	file, err := os.Open(scanFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var results ScanResults
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &results, nil
}

// loadScanResultsText loads results from text file
func loadScanResultsText(scanFile string) (*ScanResults, error) {
	content, err := os.ReadFile(scanFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	results := &ScanResults{
		References: []types.ImageReference{},
	}

	lines := strings.Split(string(content), "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse format: source_file:line_number:original_url
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid format at line %d: %s", lineNum+1, line)
		}

		var lineNumber int
		if _, err := fmt.Sscanf(parts[1], "%d", &lineNumber); err != nil {
			return nil, fmt.Errorf("invalid line number at line %d: %s", lineNum+1, parts[1])
		}

		ref := types.ImageReference{
			SourceFile:  parts[0],
			LineNumber:  lineNumber,
			OriginalURL: parts[2],
			LocalPath:   parts[2], // Will be resolved later
		}
		results.References = append(results.References, ref)
	}

	results.TotalImages = len(results.References)
	return results, nil
}

// loadScanResultsCSV loads results from CSV file
func loadScanResultsCSV(scanFile string) (*ScanResults, error) {
	file, err := os.Open(scanFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 1 {
		return nil, fmt.Errorf("empty CSV file")
	}

	results := &ScanResults{
		References: []types.ImageReference{},
	}

	// Skip header row
	for i, record := range records[1:] {
		if len(record) < 3 {
			return nil, fmt.Errorf("invalid CSV record at line %d", i+2)
		}

		var lineNumber int
		if _, err := fmt.Sscanf(record[1], "%d", &lineNumber); err != nil {
			return nil, fmt.Errorf("invalid line number at CSV line %d: %s", i+2, record[1])
		}

		ref := types.ImageReference{
			SourceFile:  record[0],
			LineNumber:  lineNumber,
			OriginalURL: record[2],
			LocalPath:   record[2], // Will be resolved later
		}
		results.References = append(results.References, ref)
	}

	results.TotalImages = len(results.References)
	return results, nil
}

// loadScanResultsAuto attempts to detect and load file format automatically
func loadScanResultsAuto(scanFile string) (*ScanResults, error) {
	// Try JSON first
	if results, err := loadScanResultsJSON(scanFile); err == nil {
		return results, nil
	}

	// Try text format
	if results, err := loadScanResultsText(scanFile); err == nil {
		return results, nil
	}

	// Try CSV
	if results, err := loadScanResultsCSV(scanFile); err == nil {
		return results, nil
	}

	return nil, fmt.Errorf("unable to detect file format for: %s", scanFile)
}
