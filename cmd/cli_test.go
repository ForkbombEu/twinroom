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
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create sample slang files
	err := os.WriteFile(filepath.Join(tempDir, "file1.slang"), []byte("contract content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1.slang: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "file2.slang"), []byte("contract content 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2.slang: %v", err)
	}

	// Prepare the command
	cmd := exec.Command("go", "run", "../main.go", "list", tempDir)

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	// Check for errors
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify the output
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
}

func TestRunCommand(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Write a sample slang file
	content := `Given nothing
Then print the string 'Hello'`
	err := os.WriteFile(filepath.Join(tempDir, "test.slang"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test.slang: %v", err)
	}

	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "../main.go", "run", tempDir, "test")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	// Check for errors
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify the output
	output := out.String()
	if !contains(output, "Running slangroom file:") {
		t.Errorf("Expected output to contain 'Running slangroom file:', got %v", output)
	}
	if !contains(output, "from folder:") {
		t.Errorf("Expected output to contain 'from folder:', got %v", output)
	}
	if !contains(output, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got %v", output)
	}
}

// Helper function to check if a substring is in a string
func contains(str, substr string) bool {
	return len(str) >= len(substr) && strings.Contains(str, substr)
}
