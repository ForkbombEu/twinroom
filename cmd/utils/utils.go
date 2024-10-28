package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	slangroom "github.com/dyne/slangroom-exec/bindings/go"
)

// LoadAdditionalData loads and validates JSON data for additional fields in SlangroomInput.
func LoadAdditionalData(path string, filename string, input *slangroom.SlangroomInput) error {
	fields := []struct {
		fieldName string
		target    *string
	}{
		{"data", &input.Data},
		{"keys", &input.Keys},
		{"extra", &input.Extra},
		{"context", &input.Context},
		{"conf", &input.Conf},
	}

	for _, field := range fields {

		jsonFile := filepath.Join(path, fmt.Sprintf("%s.%s.json", filename, field.fieldName))
		if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
			continue // Skip if file does not exist
		}

		content, err := os.ReadFile(jsonFile)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", jsonFile, err)
		}
		// Validate JSON format
		if err := validateJSON(content); err != nil {
			return fmt.Errorf("invalid JSON in %s: %w", jsonFile, err)
		}

		*field.target = string(content)
	}
	return nil
}

// validateJSON checks if the provided JSON content is well-formed.
func validateJSON(content []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(content, &temp); err != nil {
		return err
	}
	return nil
}
