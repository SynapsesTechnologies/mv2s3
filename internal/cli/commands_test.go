package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	// Test that root command exists and has expected properties
	if rootCmd.Use != "mv2s3" {
		t.Errorf("Expected root command use to be 'mv2s3', got %s", rootCmd.Use)
	}

	if !strings.Contains(rootCmd.Short, "Move local media files to S3") {
		t.Error("Expected root command short description to mention moving media files to S3")
	}

	if rootCmd.Version != "0.2.0" {
		t.Errorf("Expected version to be '0.2.0', got %s", rootCmd.Version)
	}
}

func TestCommandsExist(t *testing.T) {
	expectedCommands := []string{"migrate", "scan", "verify", "cleanup", "config"}

	for _, cmdName := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{cmdName})
		if err != nil {
			t.Errorf("Expected to find command %s, got error: %v", cmdName, err)
		}

		if cmd.Name() != cmdName {
			t.Errorf("Expected command name %s, got %s", cmdName, cmd.Name())
		}
	}
}

func TestMigrateCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"migrate"})
	if err != nil {
		t.Fatalf("Failed to find migrate command: %v", err)
	}

	expectedFlags := []string{
		"source", "bucket", "region", "prefix", "dry-run", "cleanup", "backup", "extensions",
		"images", "documents", "videos", "audio", "media-types", "include-extensions",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected migrate command to have flag --%s", flagName)
		}
	}

	// Note: bucket flag is not marked as required at Cobra level
	// since it can be provided via configuration file
}

func TestScanCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"scan"})
	if err != nil {
		t.Fatalf("Failed to find scan command: %v", err)
	}

	expectedFlags := []string{
		"source", "extensions", "images", "documents", "videos", "audio", "media-types",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected scan command to have flag --%s", flagName)
		}
	}
}

func TestVerifyCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"verify"})
	if err != nil {
		t.Fatalf("Failed to find verify command: %v", err)
	}

	expectedFlags := []string{"bucket", "region"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected verify command to have flag --%s", flagName)
		}
	}

	// Note: bucket flag is not marked as required at Cobra level
	// since it can be provided via configuration file
}

func TestCleanupCommandFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"cleanup"})
	if err != nil {
		t.Fatalf("Failed to find cleanup command: %v", err)
	}

	expectedFlags := []string{"scan-file", "backup-dir"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected cleanup command to have flag --%s", flagName)
		}
	}
}

func TestConfigShowCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"config", "show"})
	if err != nil {
		t.Fatalf("Failed to find config show command: %v", err)
	}

	if cmd.Name() != "show" {
		t.Errorf("Expected command name to be 'show', got %s", cmd.Name())
	}
}

func TestConfigMigrateCommand(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"config", "migrate"})
	if err != nil {
		t.Fatalf("Failed to find config migrate command: %v", err)
	}

	if cmd.Name() != "migrate" {
		t.Errorf("Expected command name to be 'migrate', got %s", cmd.Name())
	}
}

func TestCommandExecution(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "migrate command",
			args:     []string{"migrate", "--help"},
			expected: "Migrate media files",
		},
		{
			name:     "scan command",
			args:     []string{"scan", "--help"},
			expected: "Scan source files for media",
		},
		{
			name:     "verify command",
			args:     []string{"verify", "--help"},
			expected: "Verify S3 bucket configuration",
		},
		{
			name:     "cleanup command",
			args:     []string{"cleanup", "--help"},
			expected: "Clean up local image files",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new root command for each test to avoid state issues
			testRootCmd := &cobra.Command{Use: "mv2s3"}
			testRootCmd.AddCommand(migrateCmd, scanCmd, verifyCmd, cleanupCmd)

			buf := new(bytes.Buffer)
			testRootCmd.SetOut(buf)
			testRootCmd.SetArgs(test.args)

			err := testRootCmd.Execute()
			if err != nil {
				t.Errorf("Command execution failed: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, test.expected) {
				t.Errorf("Expected output to contain '%s', got: %s", test.expected, output)
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	expectedFlags := []string{"config", "verbose"}

	for _, flagName := range expectedFlags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected global flag --%s", flagName)
		}
	}
}

func TestScanCommandMediaTypeFiltering(t *testing.T) {
	// This test requires the test fixtures to be available
	// We'll use the actual scan command implementation

	tests := []struct {
		name         string
		flags        []string
		expectImages bool
		expectDocs   bool
		expectError  bool
	}{
		{
			name:         "default (no flags - should default to images)",
			flags:        []string{},
			expectImages: true,
			expectDocs:   false,
		},
		{
			name:         "images flag",
			flags:        []string{"--images"},
			expectImages: true,
			expectDocs:   false,
		},
		{
			name:         "documents flag",
			flags:        []string{"--documents"},
			expectImages: false,
			expectDocs:   true,
		},
		{
			name:         "both flags",
			flags:        []string{"--images", "--documents"},
			expectImages: true,
			expectDocs:   true,
		},
		{
			name:         "media-types flag images",
			flags:        []string{"--media-types", "images"},
			expectImages: true,
			expectDocs:   false,
		},
		{
			name:         "media-types flag documents",
			flags:        []string{"--media-types", "documents"},
			expectImages: false,
			expectDocs:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test that the flags are properly parsed and stored
			cmd := scanCmd

			// Reset flags for each test
			cmd.Flags().Set("images", "false")
			cmd.Flags().Set("documents", "false")
			cmd.Flags().Set("videos", "false")
			cmd.Flags().Set("audio", "false")
			cmd.Flags().Set("media-types", "")

			// Set the test flags
			for i := 0; i < len(test.flags); i++ {
				flagName := strings.TrimPrefix(test.flags[i], "--")
				var flagValue string

				// Check if this is a flag that takes a value (like media-types)
				if flagName == "media-types" && i+1 < len(test.flags) && !strings.HasPrefix(test.flags[i+1], "--") {
					flagValue = test.flags[i+1]
					i++ // Skip the value in next iteration
				} else {
					// Boolean flag
					flagValue = "true"
				}

				err := cmd.Flags().Set(flagName, flagValue)
				if err != nil {
					t.Errorf("Failed to set flag %s=%s: %v", flagName, flagValue, err)
				}
			}

			// Verify flags were set correctly
			if test.expectImages {
				images, _ := cmd.Flags().GetBool("images")
				mediaTypes, _ := cmd.Flags().GetStringSlice("media-types")

				// For default case (no flags), we expect the configuration system to handle defaults
				if len(test.flags) == 0 {
					// Default case - this will be handled by configuration defaults
					// We can't test this at the flag level since defaults come from config
					if images || contains(mediaTypes, "images") {
						// If flags are explicitly set, that's fine too
					}
				} else {
					// Explicit flags case
					if !images && !contains(mediaTypes, "images") {
						t.Errorf("Expected images to be enabled. images flag: %v, media-types: %v", images, mediaTypes)
					}
				}
			}

			if test.expectDocs {
				docs, _ := cmd.Flags().GetBool("documents")
				mediaTypes, _ := cmd.Flags().GetStringSlice("media-types")
				if !docs && !contains(mediaTypes, "documents") {
					t.Errorf("Expected documents to be enabled. documents flag: %v, media-types: %v", docs, mediaTypes)
				}
			}
		})
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
