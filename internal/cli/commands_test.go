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

	if !strings.Contains(rootCmd.Short, "Move local images to S3") {
		t.Error("Expected root command short description to mention moving images to S3")
	}

	if rootCmd.Version != "0.1.0" {
		t.Errorf("Expected version to be '0.1.0', got %s", rootCmd.Version)
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

	expectedFlags := []string{"source", "bucket", "region", "prefix", "dry-run", "cleanup", "backup", "extensions"}

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

	expectedFlags := []string{"source", "extensions"}

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

func TestCommandExecution(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "migrate command",
			args:     []string{"migrate", "--help"},
			expected: "Migrate images from local storage to S3",
		},
		{
			name:     "scan command",
			args:     []string{"scan", "--help"},
			expected: "Scan source files for image references",
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
