package utils

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/spf13/cobra"
)

// CommandMetadata contains the data from the metadata.json
type CommandMetadata struct {
	Description string `json:"description"`
	Arguments   []struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Type        string `json:"type,omitempty"`
	} `json:"arguments"`
	Options []struct {
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Default     string   `json:"default,omitempty"`
		Choices     []string `json:"choices,omitempty"`
		Env         []string `json:"env,omitempty"`
		Hidden      bool     `json:"hidden,omitempty"`
		File        bool     `json:"file,omitempty"`
		RawData     bool     `json:"rawdata,omitempty"`
		Type        string   `json:"type,omitempty"`
	} `json:"options"`
}

// FlagData contains the necessary data for a given flag
type FlagData struct {
	Choices []string
	Env     []string
	File    [2]bool
}

// Introspection contains the data coming from zenroom introspection
type Introspection map[string]struct {
	Encoding string `json:"encoding"`
	Missing  bool   `json:"missing"`
	Name     string `json:"name"`
	Zentype  string `json:"zentype"`
}

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

// loadMetadata loads the metadata JSON file for a command, if available.
func LoadMetadata(folder *embed.FS, path string) (*CommandMetadata, error) {
	var file io.ReadCloser
	var err error

	// Check if folder is nil to determine which file system to use
	if folder != nil {
		file, err = folder.Open(path)
	} else {
		file, err = os.Open(path)
	}

	if err != nil {
		if os.IsNotExist(err) {
			// Specific error message for file not found
			return nil, fmt.Errorf("metadata file not found")
		}
		// Generic error for other types of failures
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}

	defer checks(file.Close)

	var metadata CommandMetadata
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return &metadata, nil
}

// Function to normalize argument names
func NormalizeArgumentName(name string) string {
	// Remove < and > for required arguments
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	// Remove [ and ] for optional arguments
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	return name
}

// Helper function to get argument names in order from metadata
func GetArgumentNames(arguments []struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}) []string {
	names := make([]string, len(arguments))
	for i, arg := range arguments {
		names[i] = arg.Name
	}
	return names
}

// function to retrieve only the name of a flag
func GetFlagName(flagStr string) string {
	// Split the flag string by commas
	names := strings.Split(flagStr, ", ")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if strings.HasPrefix(name, "--") {
			flagParts := strings.Fields(strings.TrimPrefix(name, "--"))
			return flagParts[0]
		}
		if strings.HasPrefix(name, "-") && len(name) == 1 {
			return strings.TrimPrefix(name, "-")
		}
	}
	return ""
}

func isValidChoice(value string, choices []string) bool {
	if value == "" {
		return true
	}
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

// MergeJSON combines two JSON strings into one, with keys from the second JSON overwriting those in the first.
func MergeJSON(json1, json2 string) (string, error) {
	var map1, map2 map[string]interface{}

	if err := json.Unmarshal([]byte(json1), &map1); err != nil {
		return "", fmt.Errorf("error decoding JSON1: %v", err)
	}

	if err := json.Unmarshal([]byte(json2), &map2); err != nil {
		return "", fmt.Errorf("error decoding JSON2: %v", err)
	}

	for key, value := range map2 {
		map1[key] = value
	}

	mergedJSON, err := json.Marshal(map1)
	if err != nil {
		return "", fmt.Errorf("error encoding merged JSON: %v", err)
	}

	return string(mergedJSON), nil
}

// ConfigureArgumentsAndFlags configures the command's arguments and flags based on provided metadata,
func ConfigureArgumentsAndFlags(fileCmd *cobra.Command, metadata *CommandMetadata) (map[string]interface{}, map[string]FlagData, error) {
	argContents := make(map[string]interface{})
	flagContents := make(map[string]FlagData)

	requiredArgs := 0
	// Add arguments from metadata in the order specified
	for _, arg := range metadata.Arguments {
		argContents[NormalizeArgumentName(arg.Name)] = ""
		if strings.HasPrefix(arg.Name, "<") && strings.HasSuffix(arg.Name, ">") {
			requiredArgs++
		} else {
			fileCmd.Args = cobra.MaximumNArgs(len(metadata.Arguments))
		}
	}

	// Construct the fileCmd.Use string with all arguments
	fileCmd.Use += " " + strings.Join(GetArgumentNames(metadata.Arguments), " ")
	// Set the minimum number of required arguments
	fileCmd.Args = cobra.MinimumNArgs(requiredArgs)

	// Configure flags
	for _, opt := range metadata.Options {
		names := strings.Split(opt.Name, ", ")
		var flag, shorthand, helpText string

		// Parse through the names to extract flag, shorthand, and help text
		for _, name := range names {
			name = strings.TrimSpace(name)

			if strings.HasPrefix(name, "--") {
				flagParts := strings.Fields(strings.TrimPrefix(name, "--"))
				flag = flagParts[0]
				helpText = name
			} else if strings.HasPrefix(name, "-") {
				// Extract shorthand by removing "-" prefix
				shorthand = strings.TrimPrefix(name, "-")
			}
		}

		// Prepare the description including choices, if available
		description := opt.Description
		if len(opt.Choices) > 0 {
			description += fmt.Sprintf(" (Choices: %v)", opt.Choices)
		}
		if opt.File {
			description += ` ("-" for read from stdin)`
		}

		if opt.Default != "" {
			fileCmd.Flags().StringP(flag, shorthand, opt.Default, description)
		} else {
			fileCmd.Flags().StringP(flag, shorthand, "", description)
		}
		if opt.Hidden {
			err := fileCmd.Flags().MarkHidden(flag)
			if err != nil {
				return nil, nil, fmt.Errorf("error hiding a flag: %v", err)
			}
		}

		flagContents[flag] = FlagData{
			Choices: opt.Choices,
			Env:     opt.Env,
			File:    [2]bool{opt.File, opt.RawData},
		}

		if helpText != "" && description != "" {
			fileCmd.Flags().Lookup(flag).Usage = fmt.Sprintf("%s %s", helpText, description)
		}
	}

	return argContents, flagContents, nil
}

// ValidateFlags checks if the flag values passed to the command match any predefined choices and
// sets corresponding environment variables if specified in the flag's metadata. If a flag's value
// does not match an available choice, an error is returned.
func ValidateFlags(cmd *cobra.Command, flagContents map[string]FlagData, argContents map[string]interface{}, input *slangroom.SlangroomInput) error {
	for flag, content := range flagContents {
		var err error
		value, _ := cmd.Flags().GetString(flag)
		// Check if value should be read from stdin
		if content.File[0] {
			var fileContent []byte
			if value == "-" {
				fileContent, err = io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("error reading value for flag %s from stdin: %w", flag, err)
				}
			} else if value != "" {
				fileContent, err = os.ReadFile(value)
				if err != nil {
					return fmt.Errorf("failed to read file at path %s: %w", value, err)
				}
			}
			var jsonContent interface{}
			if !content.File[1] {
				if err := validateJSON(fileContent); err != nil {
					return fmt.Errorf("invalid JSON in %s: %w", flag, err)
				}
				if input.Data != "" {
					if input.Data, err = MergeJSON(input.Data, string(fileContent)); err != nil {
						log.Println("Error encoding arguments to JSON:", err)
						os.Exit(1)
					}
				} else {
					input.Data = string(fileContent)
				}
				value = ""
			} else {
				if err = json.Unmarshal(fileContent, &jsonContent); err == nil {
					argContents[flag] = jsonContent
					value = ""
				} else {
					value = strings.TrimSpace(string(fileContent))
				}
			}
		}
		if value == "" && len(content.Env) > 0 {
			// Try reading the value from the environment variables
			for _, envVar := range content.Env {
				envValue := os.Getenv(envVar)
				if envValue != "" {
					value = envValue
					break
				}
			}
		}
		if value != "" {
			argContents[flag] = value
		}
		if (content.Choices != nil) && !isValidChoice(value, content.Choices) {
			return fmt.Errorf("invalid input '%s' for flag: %s. Valid choices are: %v", value, flag, content.Choices)
		}
	}
	return nil
}

// map a string representing a type to the type itself
func MapTypeToGoType(typeStr string) reflect.Type {
	switch strings.ToLower(typeStr) {
	case "string":
		return reflect.TypeOf("")
	case "integer", "int":
		return reflect.TypeOf(0)
	case "number", "float", "float64":
		return reflect.TypeOf(0.0)
	case "boolean", "bool":
		return reflect.TypeOf(false)
	default:
		return reflect.TypeOf("") // Default to string if type is unknown
	}
}

// returns the default value from a given type
func CreateDefaultValue(typeStr string) interface{} {
	switch strings.ToLower(typeStr) {
	case "string":
		return ""
	case "integer", "int":
		return 0
	case "number", "float", "float64":
		return 0.0
	case "boolean", "bool":
		return false
	default:
		return "" // Default to string if type is unknown
	}
}

// Function that dynamically generates a go structure from introspection or metadata
func GenerateStruct(metadata CommandMetadata, introspectionData string) (interface{}, error) {
	var fields []reflect.StructField
	title := cases.Title(language.English)

	// If introspection data is provided, parse it and generate the struct
	if introspectionData != "" {
		var introspection Introspection
		if err := json.Unmarshal([]byte(introspectionData), &introspection); err != nil {
			return nil, fmt.Errorf("failed to parse introspection data: %v", err)
		}

		// Add fields from introspection data
		for _, info := range introspection {
			fields = append(fields, reflect.StructField{
				Name: title.String(info.Name),
				Type: MapTypeToGoType(info.Encoding),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, info.Name)),
			})
		}
	}

	// Add fields for arguments from metadata
	for _, info := range metadata.Arguments {
		name := NormalizeArgumentName(info.Name)
		fields = append(fields, reflect.StructField{
			Name: title.String(name),
			Type: MapTypeToGoType(info.Type),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
		})
	}

	// Add fields for options from metadata
	for _, opt := range metadata.Options {
		name := GetFlagName(opt.Name)
		if opt.File && !opt.RawData {
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: MapTypeToGoType(opt.Type),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" jsonschema_extras:"format=binary"`, name)),
			})
		} else {
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: MapTypeToGoType(opt.Type),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
			})
		}
	}

	// Create a new struct type
	dynamicType := reflect.StructOf(fields)

	// Create an instance of the struct
	return reflect.New(dynamicType).Interface(), nil
}

// a functionfor defer error handling
func checks(fs ...func() error) {
	for i := len(fs) - 1; i >= 0; i-- {
		if err := fs[i](); err != nil {
			log.Println("Error::", err)
		}
	}
}
