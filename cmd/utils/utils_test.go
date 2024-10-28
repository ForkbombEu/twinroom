package utils

import (
	"os"
	"path/filepath"
	"testing"

	slangroom "github.com/dyne/slangroom-exec/bindings/go"
)

// TestLoadAdditionalData tests the LoadAdditionalData function.
func TestLoadAdditionalData(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	filename := "testfile"

	// Prepare test cases
	testCases := []struct {
		name          string
		files         map[string]string
		expectedInput slangroom.SlangroomInput
		expectError   bool
	}{
		{
			name: "Valid JSON files",
			files: map[string]string{
				"testfile.data.json":    `{"string": "value"}`,
				"testfile.keys.json":    `{"key1": "value1"}`,
				"testfile.extra.json":   `{"extra": "extraValue"}`,
				"testfile.context.json": `{"context": "contextValue"}`,
				"testfile.conf.json":    `{"conf": "confValue"}`,
			},
			expectedInput: slangroom.SlangroomInput{
				Data:    `{"string": "value"}`,
				Keys:    `{"key1": "value1"}`,
				Extra:   `{"extra": "extraValue"}`,
				Context: `{"context": "contextValue"}`,
				Conf:    `{"conf": "confValue"}`,
			},
			expectError: false,
		},
		{
			name:  "Missing files",
			files: map[string]string{
				// No files created for this case
			},
			expectedInput: slangroom.SlangroomInput{},
			expectError:   false,
		},
		{
			name: "Invalid JSON file",
			files: map[string]string{
				"testfile.data.json": `{"key": "value"}`,
				"testfile.keys.json": `invalid json`,
			},
			expectedInput: slangroom.SlangroomInput{
				Data: `{"key": "value"}`,
				Keys: "",
			},
			expectError: true,
		},
	}

	// Execute test cases
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create files as per the test case
			var createdFiles []string
			for filename, content := range tt.files {
				filePath := filepath.Join(tempDir, filename)
				err := os.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				createdFiles = append(createdFiles, filePath)
			}

			input := slangroom.SlangroomInput{}

			// Call LoadAdditionalData
			err := LoadAdditionalData(tempDir, filename, &input)

			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				if !(input == tt.expectedInput) {
					t.Errorf("Expected input: %+v, got: %+v", tt.expectedInput, input)
				}
			}

			// Clean up created files
			for _, file := range createdFiles {
				if err := os.Remove(file); err != nil {
					t.Errorf("Failed to remove file %s: %v", file, err)
				}
			}
		})
	}
}

// TestValidateJSON tests the validateJSON function.
func TestValidateJSON(t *testing.T) {
	// Prepare test cases
	test := []struct {
		name        string
		content     []byte
		expectError bool
	}{
		{
			name:        "Valid JSON",
			content:     []byte(`{"key": "value"}`),
			expectError: false,
		},
		{
			name:        "Empty JSON",
			content:     []byte(`{}`),
			expectError: false,
		},
		{
			name:        "Invalid JSON - Missing quote",
			content:     []byte(`{"key": "value`),
			expectError: true,
		},
		{
			name:        "Invalid JSON - Extra comma",
			content:     []byte(`{"key": "value",}`),
			expectError: true,
		},
		{
			name:        "Invalid JSON - Incorrect structure",
			content:     []byte(`{"key": "value", "nested": { "subkey": }}`),
			expectError: true,
		},
	}

	// Execute test cases
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSON(tt.content)

			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
