package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	swagger "github.com/davidebianchi/gswagger"
	"github.com/davidebianchi/gswagger/support/gorilla"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/gemini/cmd/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
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
	okHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Fatal("unexpected error:", err)
		}
	}

	err := fouter.CreateFileRouter(input.Path, input.EmbeddedFolder, input.EmbeddedPath, func(file fouter.SlangFile) {
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
				if err != nil {
					return
				}
				dynamicStruct, _ = utils.GenerateStruct(utils.CommandMetadata{}, introspectionData)
			}
			_, err = router.AddRoute(http.MethodPost, "/"+relativePath, okHandler, swagger.Definitions{
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
				Description: strings.ReplaceAll(file.Content, "\n", "\n\n"),
			})
			if err != nil {
				return
			}
			_, err = router.AddRoute(http.MethodGet, "/"+relativePath, okHandler, swagger.Definitions{
				Tags: []string{"📑 Zencodes"},
				Querystring: func() swagger.ParameterValue {
					queryParameters := swagger.ParameterValue{}
					if metadata != nil {
						// Add arguments as query parameters
						for _, arg := range metadata.Arguments {
							name := utils.NormalizeArgumentName(arg.Name)
							queryParameters[name] = swagger.Parameter{
								Schema: &swagger.Schema{
									Value:                     utils.CreateDefaultValue(arg.Type),
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
									Value:                     utils.CreateDefaultValue(opt.Type),
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
								queryParameters[info.Name] = swagger.Parameter{
									Schema: &swagger.Schema{
										Value:                     utils.CreateDefaultValue(info.Encoding),
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
				Description: strings.ReplaceAll(file.Content, "\n", "\n\n"),
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
