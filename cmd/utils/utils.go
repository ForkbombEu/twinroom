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
	"regexp"
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
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Type        string                 `json:"type,omitempty"`
		Properties  map[string]interface{} `json:"properties,omitempty"` // For complex object types
	} `json:"arguments"`
	Options []struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Default     string                 `json:"default,omitempty"`
		Choices     []string               `json:"choices,omitempty"`
		Env         []string               `json:"env,omitempty"`
		Hidden      bool                   `json:"hidden,omitempty"`
		File        bool                   `json:"file,omitempty"`
		RawData     bool                   `json:"rawdata,omitempty"`
		Type        string                 `json:"type,omitempty"`
		Properties  map[string]interface{} `json:"properties,omitempty"` // For complex object types
	} `json:"options"`
}

// FlagData contains the necessary data for a given flag
type FlagData struct {
	Choices []string
	Env     []string
	File    [2]bool
}
type codec struct {
	Encoding string `json:"encoding"`
	Missing  bool   `json:"missing"`
	Name     string `json:"name"`
	Zentype  string `json:"zentype"`
}

// Introspection contains the data coming from zenroom introspection
type Introspection map[string]codec

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
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
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
func ConfigureArgumentsAndFlags(fileCmd *cobra.Command, metadata *CommandMetadata, introspectionData string) (map[string]interface{}, map[string]FlagData, error) {
	argContents := make(map[string]interface{})
	flagContents := make(map[string]FlagData)

	requiredArgs := 0
	// Add arguments from metadata in the order specified
	if metadata != nil {
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
	}
	if introspectionData != "" {
		var introspection Introspection
		if err := json.Unmarshal([]byte(introspectionData), &introspection); err != nil {
			return argContents, flagContents, fmt.Errorf("failed to parse introspection data: %v", err)
		}

		// Add fields from introspection data
		for _, info := range introspection {
			fileCmd.Flags().StringP(info.Name, "", "", "flag addeded from introspection")
			flagContents[info.Name] = FlagData{}
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
func MapTypeToGoType(typeStr string, elemTypeStr string) reflect.Type {
	switch strings.ToLower(typeStr) {
	case "string":
		return reflect.TypeOf("")
	case "integer", "int":
		return reflect.TypeOf(0)
	case "number", "float", "float64":
		return reflect.TypeOf(0.0)
	case "boolean", "bool":
		return reflect.TypeOf(false)
	case "array", "[]":
		elemType := MapTypeToGoType(elemTypeStr, "") // Recursively map element type
		return reflect.SliceOf(elemType)
	case "dictionary", "map", "object":
		elemType := MapTypeToGoType(elemTypeStr, "")
		return reflect.MapOf(reflect.TypeOf(""), elemType)
	default:
		return reflect.TypeOf("") // Default to string if type is unknown
	}
}

// returns the default value from a given type
func CreateDefaultValue(typeStr string, elemTypeStr string, nestedFields ...reflect.StructField) interface{} {
	switch strings.ToLower(typeStr) {
	case "string":
		return ""
	case "integer", "int":
		return 0
	case "number", "float", "float64":
		return 0.0
	case "boolean", "bool":
		return false
	case "array", "[]":
		elemType := CreateDefaultValue(elemTypeStr, "") // Recursively get default value for array element type
		return reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(elemType)), 0, 0).Interface()
	case "dictionary", "map", "object":
		if len(nestedFields) > 0 {
			// Handle nested object fields by creating a struct
			structFields := []reflect.StructField{}
			for _, field := range nestedFields {
				structFields = append(structFields, reflect.StructField{
					Name: field.Name,
					Type: field.Type,
					Tag:  field.Tag,
				})
			}
			// Create a struct and populate it with default values for each field
			structType := reflect.StructOf(structFields)
			structValue := reflect.New(structType).Elem()

			// Set default values for the struct fields
			for i := 0; i < structType.NumField(); i++ {
				field := structType.Field(i)
				defaultValue := CreateDefaultValue(field.Type.String(), "", nestedFields...)
				structValue.Field(i).Set(reflect.ValueOf(defaultValue))
			}

			return structValue.Interface()
		}
		// If no nested fields, return a default empty struct
		return reflect.New(reflect.StructOf([]reflect.StructField{})).Elem().Interface()
	default:
		return "" // Default to string if type is unknown
	}
}

// Coverts zentype to type
func ZentypeToType(codec codec) string {
	switch codec.Zentype {
	case "a":
		return "array"
	case "d":
		return "object"
	default:
		return codec.Encoding
	}
}

// Helper function to handle nested objects from metadata.
func ParseObjectProperties(properties map[string]interface{}) []reflect.StructField {
	var nestedFields []reflect.StructField
	title := cases.Title(language.English)

	for name, value := range properties {
		fieldName := title.String(name)
		switch v := value.(type) {
		case map[string]interface{}:
			if v["type"] == "object" {
				// Recursively parse nested objects
				if subProperties, ok := v["properties"].(map[string]interface{}); ok {
					nestedFields = append(nestedFields, reflect.StructField{
						Name: fieldName,
						Type: reflect.StructOf(ParseObjectProperties(subProperties)),
						Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
					})
				}
			} else {
				// Handle primitive types
				nestedFields = append(nestedFields, reflect.StructField{
					Name: fieldName,
					Type: MapTypeToGoType(v["type"].(string), ""),
					Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
				})
			}
		default:
			// Invalid structure
			fmt.Printf("Warning: invalid property structure for %s\n", name)
		}
	}
	return nestedFields
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
			typeStr := ZentypeToType(info)
			fields = append(fields, reflect.StructField{
				Name: title.String(info.Name),
				Type: MapTypeToGoType(typeStr, info.Encoding),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, info.Name)),
			})
		}
	}

	// Add fields for arguments from metadata
	for _, arg := range metadata.Arguments {
		name := NormalizeArgumentName(arg.Name)
		if arg.Type == "object" && arg.Properties != nil {
			// Nested object handling
			nestedFields := ParseObjectProperties(arg.Properties)
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: reflect.StructOf(nestedFields),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
			})
		} else {
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: MapTypeToGoType(arg.Type, "string"),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
			})
		}
	}

	// Add fields for options from metadata
	for _, opt := range metadata.Options {
		name := GetFlagName(opt.Name)
		if opt.Type == "object" && opt.Properties != nil {
			// Nested object handling
			nestedFields := ParseObjectProperties(opt.Properties)
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: reflect.StructOf(nestedFields),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
			})
		} else if opt.File && !opt.RawData {
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: MapTypeToGoType(opt.Type, ""),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" jsonschema_extras:"format=binary"`, name)),
			})
		} else {
			var choices string
			for i, choice := range opt.Choices {
				if i > 0 {
					choices += ","
				}
				choices += "enum=" + choice
			}
			if opt.Default != "" {
				if choices != "" {
					choices += ","
				}
				choices += "default=" + opt.Default
			}
			fields = append(fields, reflect.StructField{
				Name: title.String(name),
				Type: MapTypeToGoType(opt.Type, ""),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" jsonschema:"%s"`, name, choices)),
			})
		}
	}

	// Create the final struct
	structType := reflect.StructOf(fields)
	return reflect.New(structType).Interface(), nil
}

// a functionfor defer error handling
func checks(fs ...func() error) {
	for i := len(fs) - 1; i >= 0; i-- {
		if err := fs[i](); err != nil {
			log.Println("Error::", err)
		}
	}
}

// isDir checks if a given path is a directory
func IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err // Error, e.g., the path doesn't exist.
	}
	return info.IsDir(), nil
}

// Utils function that removes from introspection JSON the data generated by slangroom
func CleanIntrospection(inputStr, jsonStr string) string {
	// Find all names between single quotes after "output into" or "output as"
	re := regexp.MustCompile(`(?:output into|output as) '([^']+)'`)
	matches := re.FindAllStringSubmatch(inputStr, -1)

	// Extract the names from matches
	var names []string
	for _, match := range matches {
		if len(match) > 1 {
			names = append(names, match[1])
		}
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}
	removeKeys(data, names)
	cleanedJSON, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return string(cleanedJSON)
}

// removeKeys removes keys from the map if they match the provided names.
func removeKeys(data map[string]interface{}, names []string) {
	for _, name := range names {
		delete(data, name)
	}

	// Recursively clean nested maps
	for _, value := range data {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			removeKeys(nestedMap, names)
		}
	}
}
