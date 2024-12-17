package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	swagger "github.com/davidebianchi/gswagger"
	"github.com/davidebianchi/gswagger/support/gorilla"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/twinroom/cmd/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/invopop/jsonschema"
	jsschema "github.com/santhosh-tekuri/jsonschema/v5"
)

type errorResponse struct {
	Message []string `json:"message"`
}

type outputResponse struct {
	Output []string `json:"output"`
}

// GenerateOpenAPIRouter generates an OpenAPI router with routes defined based on slangroom contracts.
func GenerateOpenAPIRouter(ctx context.Context, input HTTPInput) (*mux.Router, error) {
	muxRouter := mux.NewRouter()
	router, _ := swagger.NewRouter(gorilla.NewRouter(muxRouter), swagger.Options{
		Context: ctx,
		Openapi: &openapi3.T{
			Info: &openapi3.Info{
				Title:   input.BinaryName,
				Version: "1.0.0",
			},
			Tags: openapi3.Tags{
				{
					Name:        "📑 Zencodes",
					Description: "Endpoints generated over the Zencode smart contracts",
				},
			},
		},
	})
	folderPath := input.EmbeddedPath
	if input.EmbeddedSubDir != "" {
		folderPath = input.EmbeddedPath + "/" + input.EmbeddedSubDir
	}
	err := fouter.CreateFileRouter(input.Path, input.EmbeddedFolder, folderPath, func(file fouter.SlangFile) {
		var filename string
		if input.FileName == "" {
			filename = strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))
		} else {
			filename = input.FileName
		}
		if filename == strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName)) {
			var err error
			var relativePath string
			if input.EmbeddedPath != "" {
				relativePath = strings.TrimPrefix(filepath.Join(file.Dir, file.FileName), input.EmbeddedPath+"/")
				relativePath = strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
			} else {
				relativePath = strings.TrimSuffix(filepath.Join(file.Dir, file.FileName), filepath.Ext(file.FileName))
			}
			metadataPath := filepath.Join(file.Dir, filename+".metadata.json")
			var dynamicStruct interface{}
			var introspectionData string
			metadata, err := utils.LoadMetadata(input.EmbeddedFolder, metadataPath)
			if err != nil && err.Error() != "metadata file not found" {
				log.Printf("WARNING: error in metadata for contracts: %s\n", file.FileName)
				log.Println(err)
			} else if err == nil {
				dynamicStruct, _ = utils.GenerateStruct(*metadata, "")
			} else {
				introspectionData, err = slangroom.Introspect(file.Content)
				if err == nil {
					data, cleanErr := utils.CleanIntrospection(file.Content, introspectionData)
					if err != nil {
						log.Printf("unexpected error during introspection: %v", cleanErr)
					}
					introspectionData = data
				} else {
					introspectionData = ""
				}
				dynamicStruct, _ = utils.GenerateStruct(utils.CommandMetadata{}, introspectionData)
			}
			_, err = router.AddRoute(http.MethodPost, "/"+relativePath, gorilla.HandlerFunc(createSlangroomHandler(file, dynamicStruct)), swagger.Definitions{
				Tags: []string{"📑 Zencodes"},
				RequestBody: &swagger.ContentValue{
					Content: swagger.Content{
						"application/json": {Value: dynamicStruct, AllowAdditionalProperties: true},
					},
				},
				Responses: map[int]swagger.ContentValue{
					200: {
						Content: swagger.Content{
							"application/json": {Value: &outputResponse{}, AllowAdditionalProperties: true},
						},
						Description: "The slangroom execution output, split by newline",
					},
					500: {
						Content: swagger.Content{
							"application/json": {Value: &errorResponse{}, AllowAdditionalProperties: true},
						},
						Description: "Slangroom execution error",
					},
				},
				Description: file.Content,
			})
			if err != nil {
				return
			}
			_, err = router.AddRoute(http.MethodGet, "/"+relativePath, gorilla.HandlerFunc(createSlangroomHandler(file, nil)), swagger.Definitions{
				Tags: []string{"📑 Zencodes"},
				Querystring: func() swagger.ParameterValue {
					queryParameters := swagger.ParameterValue{}
					if metadata != nil {
						// Add arguments as query parameters
						for _, arg := range metadata.Arguments {
							name := utils.NormalizeArgumentName(arg.Name)
							queryParameters[name] = swagger.Parameter{
								Schema: &swagger.Schema{
									Value:                     utils.CreateDefaultValue(arg.Type, "", utils.ParseObjectProperties(arg.Properties)...),
									AllowAdditionalProperties: true,
								},
								Description: arg.Description,
							}
						}

						// Add options as query parameters
						for _, opt := range metadata.Options {
							name := utils.GetFlagName(opt.Name)
							queryParameters[name] = swagger.Parameter{
								Schema: &swagger.Schema{
									Value:                     utils.CreateDefaultValue(opt.Type, "", utils.ParseObjectProperties(opt.Properties)...),
									AllowAdditionalProperties: true,
								},
								Description: opt.Description,
							}
						}
					}
					if introspectionData != "" {
						var introspection utils.Introspection
						if err := json.Unmarshal([]byte(introspectionData), &introspection); err == nil {
							for _, info := range introspection {
								typeStr := utils.ZentypeToType(info)
								queryParameters[info.Name] = swagger.Parameter{
									Schema: &swagger.Schema{
										Value:                     utils.CreateDefaultValue(typeStr, info.Encoding),
										AllowAdditionalProperties: true,
									},
									Description: fmt.Sprintf("The %s", info.Name),
								}
							}
						}
					}
					return queryParameters
				}(),
				Responses: map[int]swagger.ContentValue{
					200: {
						Content: swagger.Content{
							"application/json": {Value: &outputResponse{}, AllowAdditionalProperties: true},
						},
						Description: "The slangroom execution output, splitted by newline",
					},
					500: {
						Content: swagger.Content{
							"application/json": {Value: &errorResponse{}, AllowAdditionalProperties: true},
						},
						Description: "Slangroom execution error",
					},
				},
				Description: file.Content,
			})
			if err != nil {
				return
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("error creating file router: %v", err)
	}

	// Expose OpenAPI documentation
	err = router.GenerateAndExposeOpenapi()
	if err != nil {
		return nil, fmt.Errorf("error creating opeanpapi spec: %v", err)
	}
	return muxRouter, nil
}
func getQueryParams(r *http.Request) map[string]interface{} {
	data := make(map[string]interface{})

	// Get the query string from the URL
	q := r.URL.Query()

	// Iterate over the query parameters
	for key, values := range q {
		// Use the first value if there are multiple values for the same key
		if len(values) > 0 {
			data[key] = values[0]
		}
	}

	return data
}

func createSlangroomHandler(file fouter.SlangFile, dynamicStruct interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleSlangroomRequest(file, dynamicStruct, w, r)
	}
}

func handleSlangroomRequest(file fouter.SlangFile, dynamicStruct interface{}, w http.ResponseWriter, r *http.Request) {
	var input map[string]interface{}

	// Handle POST request with JSON body
	if r.Method == http.MethodPost {
		if r.Body == nil || r.ContentLength == 0 {
			// If body is empty, initialize input with default or empty values
			input = make(map[string]interface{})
		} else {
			// Read and buffer the request body for multiple decodes
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusInternalServerError)
				return
			}

			// Decode into dynamicStruct for validation
			if err := ValidateJSONAgainstStruct(bodyBytes, dynamicStruct); err != nil {
				http.Error(w, fmt.Sprintf("Invalid JSON payload for validation: %v", err), http.StatusInternalServerError)
				return
			}
			// Decode into a generic map for further processing
			if err := json.Unmarshal(bodyBytes, &input); err != nil {
				http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	// Handle GET request with query parameters
	if r.Method == http.MethodGet {
		input = getQueryParams(r)
	}

	data, err := json.Marshal(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal input: %v", err), http.StatusInternalServerError)
		return
	}

	slangroomInput := slangroom.SlangroomInput{
		Contract: file.Content,
		Data:     string(data),
	}

	// Execute the slangroom contract
	output, err := slangroom.Exec(slangroomInput)
	if err != nil {
		log.Printf("Execution error for file %s: %v", file.FileName, output.Logs)
		http.Error(w, fmt.Sprintf("Execution error: %v", output.Logs), http.StatusInternalServerError)
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(output.Output), &jsonData); err != nil {
		log.Printf("Error parsing output as JSON: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON in output: %s", output.Output), http.StatusInternalServerError)
		return
	}

	// Format the JSON with indentation
	w.Header().Set("Content-Type", "application/json")
	formattedOutput, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Printf("Error formatting response: %v", err)
		http.Error(w, "Failed to format response", http.StatusInternalServerError)
		return
	}

	// Write the formatted JSON to the response writer
	if _, err := w.Write(formattedOutput); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// Validate the request body against the json schema
func ValidateJSONAgainstStruct(data []byte, schemaStruct interface{}) error {
	// Generate JSON schema from the struct
	schema := jsonschema.Reflect(schemaStruct)
	// Marshal schema to JSON
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Compile the schema using jsonschema/v5
	compiler := jsschema.NewCompiler()
	if err := compiler.AddResource("schema.json", bytes.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiledSchema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Unmarshal JSON data into a map[string]interface{} for validation
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	// Validate the input JSON data
	if err := compiledSchema.Validate(jsonData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
