package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/spf13/cobra"
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
				err := os.WriteFile(filePath, []byte(content), 0600)
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

func TestLoadMetadata(t *testing.T) {
	sampleMetadata := CommandMetadata{
		Description: "Test description",
		Arguments: []struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{Name: "arg1", Description: "Argument 1 description"},
			{Name: "arg2", Description: "Argument 2 description", Type: "integer"},
		},
		Options: []struct {
			Name        string   `json:"name"`
			Description string   `json:"description,omitempty"`
			Default     string   `json:"default,omitempty"`
			Choices     []string `json:"choices,omitempty"`
			Env         []string `json:"env,omitempty"`
			Hidden      bool     `json:"hidden,omitempty"`
			File        bool     `json:"file,omitempty"`
			RawData     bool     `json:"rawdata,omitempty"`
			Type        string   `json:"type,omitempty"`
		}{
			{Name: "--option1, -o", Description: "Option 1 description", Default: "default1", Choices: []string{"choice1", "choice2"}, Type: "string"},
		},
	}
	// Create a temporary directory
	tempDir := t.TempDir()

	// Define metadata file path
	metadataPath := filepath.Join(tempDir, "test_metadata.json")

	// Write the sample metadata to a temporary file
	fileContent, err := json.Marshal(sampleMetadata)
	if err != nil {
		t.Fatalf("Failed to marshal sample metadata: %v", err)
	}

	if err := os.WriteFile(metadataPath, fileContent, 0600); err != nil {
		t.Fatalf("Failed to write metadata file: %v", err)
	}

	// Run LoadMetadata using the temp file
	metadata, err := LoadMetadata(nil, metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Verify that the loaded metadata matches the sample metadata
	if metadata.Description != sampleMetadata.Description {
		t.Errorf("Expected description %s, got %s", sampleMetadata.Description, metadata.Description)
	}
	if len(metadata.Arguments) != len(sampleMetadata.Arguments) {
		t.Errorf("Expected %d arguments, got %d", len(sampleMetadata.Arguments), len(metadata.Arguments))
	}
	if len(metadata.Options) != len(sampleMetadata.Options) {
		t.Errorf("Expected %d options, got %d", len(sampleMetadata.Options), len(metadata.Options))
	}
}
func TestMergeJSON(t *testing.T) {
	tests := []struct {
		name          string
		json1         string
		json2         string
		expected      string
		expectedError bool
	}{
		{
			name:          "Merging two valid JSONs",
			json1:         `{"key1": "value1"}`,
			json2:         `{"key2": "value2"}`,
			expected:      `{"key1":"value1","key2":"value2"}`,
			expectedError: false,
		},
		{
			name:          "Merging two valid JSONs with same keys",
			json1:         `{"key1": "value1", "key3": "value3"}`,
			json2:         `{"key2": "value2", "key3": "pass"}`,
			expected:      `{"key1":"value1","key2":"value2","key3":"pass"}`,
			expectedError: false,
		},
		{
			name:          "Merging with an invalid JSON",
			json1:         `{"key1": "value1"}`,
			json2:         `{"key2": value2}`,
			expected:      "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeJSON(tt.json1, tt.json2)
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
			if !tt.expectedError && strings.TrimSpace(result) != tt.expected {
				t.Errorf("Expected result: %s, got: %s", tt.expected, result)
			}
		})
	}
}

// TestConfigureArgumentsAndFlags tests ConfigureArgumentsAndFlags function.
func TestConfigureArgumentsAndFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "testcmd",
	}

	metadata := &CommandMetadata{
		Description: "Test command",
		Arguments: []struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			Type        string `json:"type,omitempty"`
		}{
			{Name: "<arg1>", Description: "Required argument"},
			{Name: "[arg2]", Description: "Optional argument"},
		},
		Options: []struct {
			Name        string   `json:"name"`
			Description string   `json:"description,omitempty"`
			Default     string   `json:"default,omitempty"`
			Choices     []string `json:"choices,omitempty"`
			Env         []string `json:"env,omitempty"`
			Hidden      bool     `json:"hidden,omitempty"`
			File        bool     `json:"file,omitempty"`
			RawData     bool     `json:"rawdata,omitempty"`
			Type        string   `json:"type,omitempty"`
		}{
			{Name: "--flag1", Description: "Test flag", Default: "default_value"},
		},
	}

	args, flags, err := ConfigureArgumentsAndFlags(cmd, metadata)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(args) != 2 || len(flags) != 1 {
		t.Errorf("Expected 2 arguments and 1 flag, got %d arguments and %d flags", len(args), len(flags))
	}

	// Verify that flags and arguments are correctly set up
	if _, exists := flags["flag1"]; !exists {
		t.Errorf("Expected flag 'flag1' to be set in flagContents")
	}
	if _, exists := args["arg1"]; !exists {
		t.Errorf("Expected argument 'arg1' to be set in argContents")
	}
}

// TestValidateFlags tests the ValidateFlags function.
func TestValidateFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("flag1", "", "")
	cmd.Flags().String("flag2", "", "")
	cmd.Flags().String("envFlag", "", "")
	cmd.Flags().String("fileFlag", "", "")

	flagContents := map[string]FlagData{
		"flag1": {
			Choices: []string{"opt1", "opt2"},
		},
		"flag2": {
			Choices: []string{"opt1", "opt2"},
		},
		"envFlag": {
			Env: []string{"TEST_FLAG_ENV_VAR"},
		},
		"fileFlag": {
			File: [2]bool{true, true},
		},
	}

	argContents := make(map[string]interface{})

	// Test for invalid choice
	err := cmd.Flags().Set("flag2", "invalid_choice")
	if err != nil {
		t.Errorf("Unexpected error setting test flag: %v", err)
	}
	err = ValidateFlags(cmd, flagContents, argContents, nil)
	if err == nil {
		t.Errorf("Expected error for invalid flag choice, got: nil")
	}

	// Test for valid choice and check environment variable setting
	err = cmd.Flags().Set("flag1", "opt1")
	if err != nil {
		t.Errorf("Unexpected error setting test flag: %v", err)
	}
	err = cmd.Flags().Set("flag2", "opt2")
	if err != nil {
		t.Errorf("Unexpected error setting test flag: %v", err)
	}
	err = os.Setenv("TEST_FLAG_ENV_VAR", "test")
	if err != nil {
		t.Errorf("Unexpected error setting test env variable: %v", err)
	}
	err = ValidateFlags(cmd, flagContents, argContents, nil)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check if the environment variable was set correctly
	envVarValue := argContents["envFlag"]
	if envVarValue != "test" {
		t.Errorf("Expected TEST_ENV_VAR to be 'opt1', got: %v", envVarValue)
	}

	// Test reading from stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("error creating pipe: %v", err)
	}
	defer checks(r.Close)

	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()
	os.Stdin = r

	_, err = w.Write([]byte("input from stdin"))
	if err != nil {
		t.Errorf("Error writing to stdin pipe: %v", err)
	}
	checks(w.Close)

	err = cmd.Flags().Set("fileFlag", "-")
	if err != nil {
		t.Errorf("Unexpected error setting test flag: %v", err)
	}
	err = ValidateFlags(cmd, flagContents, argContents, nil)
	if err != nil {
		t.Errorf("Expected no error for stdin read, got: %v", err)
	}

	if argContents["fileFlag"] != "input from stdin" {
		t.Errorf("Expected 'input from stdin' for fileFlag, got: %v", argContents["fileFlag"])
	}

	// Test reading raw data from a file path
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}
	defer func() {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			t.Fatalf("error removing temp file: %v", err)
		}
	}()
	// Write some content to the file
	_, err = tmpFile.Write([]byte("content from file"))
	if err != nil {
		t.Fatalf("error writing to temp file: %v", err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}

	// Set the fileFlag to the path of the temporary file
	err = cmd.Flags().Set("fileFlag", tmpFile.Name())
	if err != nil {
		t.Errorf("Unexpected error setting test flag: %v", err)
	}
	err = ValidateFlags(cmd, flagContents, argContents, nil)
	if err != nil {
		t.Errorf("Expected no error for file read, got: %v", err)
	}
	if argContents["fileFlag"] != "content from file" {
		t.Errorf("Expected 'content from file' for fileFlag, got: %v", argContents["fileFlag"])
	}
	//test reading from a json

	cmd.Flags().String("jsonFlag", "", "")
	flagContents["jsonFlag"] = FlagData{
		File: [2]bool{true, false},
	}
	input := slangroom.SlangroomInput{}
	jsonFile, err := os.CreateTemp("", "testfile.json")
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}
	defer func() {
		err := os.Remove(jsonFile.Name())
		if err != nil {
			t.Fatalf("error removing temp file: %v", err)
		}
	}()

	expected := `{
    "test": {
        "name": "Myname",
        "data": "somecontent"
    },
    "array": [
        "value1",
        "value2"
    ]
}`
	// Write some content to the file
	_, err = jsonFile.Write([]byte(expected))
	if err != nil {
		t.Fatalf("error writing to temp file: %v", err)
	}
	err = jsonFile.Close()
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}

	// Set the jsonFlag to the path of the temporary file
	err = cmd.Flags().Set("jsonFlag", jsonFile.Name())
	if err != nil {
		t.Errorf("Unexpected error setting test flag: %v", err)
	}
	err = ValidateFlags(cmd, flagContents, argContents, &input)
	if err != nil {
		t.Errorf("Expected no error for file read, got: %v", err)
	}

	if input.Data != expected {
		t.Errorf("Expected %s for jsonFlag, got: %v", expected, input.Data)
	}

}
