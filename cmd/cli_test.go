package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {

	// Test listing embedded files
	t.Run("List Embedded Files", func(t *testing.T) {
		// Prepare the command without any arguments
		cmd := exec.Command("go", "run", "../main.go", "list")

		// Capture the output
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()

		// Check for errors
		if err != nil {
			t.Fatalf("Command execution failed: %v", err)
		}

		// Verify the output for embedded files
		output := out.String()
		if !contains(output, "Listing embedded slangroom files:") {
			t.Errorf("Expected output to contain 'Listing embedded slangroom files:', got %v", output)
		}
		if !contains(output, "Found file: hello.slang") {
			t.Errorf("Expected output to contain 'Found file: hello.slang', got %v", output)
		}
		if !contains(output, "Found file: test.slang") {
			t.Errorf("Expected output to contain 'Found file: test.slang', got %v", output)
		}
	})

	// Test listing files in the provided folder
	t.Run("List Files in Folder", func(t *testing.T) {
		// Create a temporary directory
		tempDir := t.TempDir()

		// Create sample slangroom files for the folder
		err := os.WriteFile(filepath.Join(tempDir, "file1.slang"), []byte("contract content 1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file1.slang: %v", err)
		}

		err = os.WriteFile(filepath.Join(tempDir, "file2.slang"), []byte("contract content 2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file2.slang: %v", err)
		}
		// Prepare the command to list files in the tempDir
		cmd := exec.Command("go", "run", "../main.go", "list", tempDir)

		// Capture the output
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()

		// Check for errors
		if err != nil {
			t.Fatalf("Command execution failed: %v", err)
		}

		// Verify the output for files in the folder
		output := out.String()
		if !contains(output, "Listing slangroom files in folder:") {
			t.Errorf("Expected output to contain 'Listing slangroom files in folder:', got %v", output)
		}
		if !contains(output, "Found file: file1.slang") {
			t.Errorf("Expected output to contain 'Found file: file1.slang', got %v", output)
		}
		if !contains(output, "Found file: file2.slang") {
			t.Errorf("Expected output to contain 'Found file: file2.slang', got %v", output)
		}
	})
}

func TestRunCommand(t *testing.T) {
	t.Run("Filesystem", func(t *testing.T) {
		tempDir := t.TempDir()

		// Write a sample slangroom file in the filesystem directory
		content := `Given nothing
Then print the string 'Hello'`
		err := os.WriteFile(filepath.Join(tempDir, "test.slang"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test.slang: %v", err)
		}
		cmd := exec.Command("go", "run", "../main.go", tempDir, "test")

		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()

		if err != nil {
			t.Fatalf("Command execution failed: %v", err)
		}

		output := out.String()
		if !contains(output, "Hello") {
			t.Errorf("Expected output to contain 'Hello', got %v", output)
		}
	})

	// Subtest for running a slangroom file from embedded files
	t.Run("Embedded", func(t *testing.T) {
		cmd := exec.Command("go", "run", "../main.go", "test", "hello")

		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			t.Fatalf("Command execution failed: %v", err)
		}
		output := out.String()
		if !contains(output, "Hello_from_embedded!") {
			t.Errorf("Expected output to contain 'Hello from embedded!', got %v", output)
		}
	})
}

// Helper function to check if a substring is in a string
func contains(str, substr string) bool {
	return len(str) >= len(substr) && strings.Contains(str, substr)
}
