package httpserver

import (
	"context"
	"embed"
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

type Introspection map[string]struct {
	Encoding string `json:"encoding"`
	Missing  bool   `json:"missing"`
	Name     string `json:"name"`
	Zentype  string `json:"zentype"`
}

type errorResponse struct {
	Message []string `json:"message"`
}

type outputResponse struct {
	Output []string `json:"output"`
}

// GenerateOpenAPIRouter generates an OpenAPI router with routes defined based on slangroom contracts.
func GenerateOpenAPIRouter(ctx context.Context, contracts *embed.FS, basePath string) (*mux.Router, error) {
	muxRouter := mux.NewRouter()
	router, _ := swagger.NewRouter(gorilla.NewRouter(muxRouter), swagger.Options{
		Context: ctx,
		Openapi: &openapi3.T{
			Info: &openapi3.Info{
				Title:   "Slangroom API",
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

	err := fouter.CreateFileRouter("", contracts, basePath, func(file fouter.SlangFile) {
		relativePath := strings.TrimPrefix(filepath.Join(file.Dir, file.FileName), basePath+"/")
		relativePath = strings.TrimSuffix(relativePath, filepath.Ext(relativePath))

		metadataPath := filepath.Join(file.Dir, strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))+".metadata.json")
		metadata, err := utils.LoadMetadata(contracts, metadataPath)
		var dynamicStruct interface{}
		var introspectionData string
		if err != nil && err.Error() != "metadata file not found" {
			log.Printf("WARNING: error in metadata for contracts: %s\n", file.FileName)
			log.Println(err)
		} else if err == nil {
			dynamicStruct, _ = utils.GenerateStruct(*metadata, "")
		} else {
			introspectionData, err = slangroom.Introspect(file.Content)
			if err != nil {
				log.Fatal("Unexpected error during introspection:", err)
			}
			dynamicStruct, _ = utils.GenerateStruct(utils.CommandMetadata{}, introspectionData)
		}
		router.AddRoute(http.MethodPost, "/"+relativePath, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}, swagger.Definitions{
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
			Description: fmt.Sprintf("<pre>%s</pre>", file.Content),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("error creating file router: %v", err)
	}

	// Expose OpenAPI documentation
	router.GenerateAndExposeOpenapi()

	return muxRouter, nil
}
