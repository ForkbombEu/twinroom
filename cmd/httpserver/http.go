package httpserver

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/gemini/cmd/utils"
)

// listSlangFilesHandler returns an HTTP handler that lists available slangroom files in the provided directories.
// It generates an HTML page displaying the slangroom files for each directory.
func listSlangFilesHandler(slangFiles map[string][]string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprintln(w, "<html><body><h1>Available contract files:</h1>")
		if err != nil {
			fmt.Printf("Failed to write http page: %v\n", err)
		}
		for dir, files := range slangFiles {
			_, err = fmt.Fprintf(w, "<h2>Directory: %s</h2><ul>", dir)
			if err != nil {
				fmt.Printf("Failed to write http page: %v\n", err)
			}
			for _, file := range files {
				fileName := filepath.Base(file)
				link := fmt.Sprintf("/slang/%s", file)
				_, err = fmt.Fprintf(w, `<li><a href="%s">%s/%s</a></li>`, link, dir, fileName)
				if err != nil {
					fmt.Printf("Failed to write http page: %v\n", err)
				}
			}

			_, err = fmt.Fprintln(w, "</ul>")
			if err != nil {
				fmt.Printf("Failed to write http page: %v\n", err)
			}
		}

		_, err = fmt.Fprintln(w, "</body></html>")
		if err != nil {
			fmt.Printf("Failed to write http page: %v\n", err)
		}
	}
}

// slangFilePageHandler returns an HTTP handler that displays the contents of a single slangroom file.
// It generates an HTML page showing the content of the file along with a button to execute it.
func slangFilePageHandler(file fouter.SlangFile) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		name := strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprintf(w, "<html><body><h1>%s</h1><pre>%s</pre>", name, file.Content)
		if err != nil {
			fmt.Printf("Failed to write http page: %v\n", err)
		}
		relativePath := filepath.Join(file.Dir, name)
		_, err = fmt.Fprintf(w, `<form method="POST" action="/slang/execute/%s">
                            <button type="submit">Execute %s</button>
                        </form>`, relativePath, filepath.Base(file.Path))
		if err != nil {
			fmt.Printf("Failed to write http page: %v\n", err)
		}
		_, err = fmt.Fprintln(w, "</body></html>")
		if err != nil {
			fmt.Printf("Failed to write http page: %v\n", err)
		}
	}
}

// executeSlangFileHandler returns an HTTP handler that executes a slangroom file via a POST request.
// The handler responds with a JSON output of the result or an error if the execution fails.
func executeSlangFileHandler(file fouter.SlangFile, baseFolder string, executionData *slangroom.SlangroomInput) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}
		var input slangroom.SlangroomInput
		if executionData == nil {
			input.Contract = file.Content

			// Load additional data from JSON files with matching names
			filename := strings.TrimSuffix(file.FileName, ".slang")
			err := utils.LoadAdditionalData(filepath.Join(baseFolder, file.Dir), filename, &input)
			if err != nil {
				http.Error(w, "Error loading additional data: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			input = *executionData
		}

		result, err := slangroom.Exec(input)
		if err != nil {
			http.Error(w, "Error executing slang file: "+result.Logs, http.StatusInternalServerError)
			return
		}

		output := map[string]interface{}{
			"output": result.Output,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(output); err != nil {
			http.Error(w, "Error encoding output to JSON", http.StatusInternalServerError)
		}
	}
}

// startHTTPServer starts an HTTP server on port 3000 to serve slangroom files from the specified folder.
func StartHTTPServer(folder string, filePath string, input *slangroom.SlangroomInput, contracts *embed.FS, embeddedPath string) error {
	ctx := context.Background()

	// Generate OpenAPI router
	mainRouter, err := GenerateOpenAPIRouter(ctx, contracts, embeddedPath)
	if err != nil {
		return fmt.Errorf("error generating OpenAPI router: %v", err)
	}

	slangFiles := make(map[string][]string)
	err = fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
		relativePath := filepath.Join(file.Dir, strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName)))
		slangFiles[file.Dir] = append(slangFiles[file.Dir], relativePath)
		mainRouter.HandleFunc("/slang/"+relativePath, slangFilePageHandler(file)).Methods("GET")
		var executionData *slangroom.SlangroomInput
		if filePath == relativePath && input != nil {
			executionData = input
		}
		mainRouter.HandleFunc("/slang/execute/"+relativePath, executeSlangFileHandler(file, folder, executionData)).Methods("POST")
	})
	if err != nil {
		return fmt.Errorf("error setting up file router: %v", err)
	}

	// Add a route for listing slang files
	mainRouter.HandleFunc("/slang/", listSlangFilesHandler(slangFiles)).Methods("GET")

	// Start server on port 3000
	port := "3000"
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		listener, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return fmt.Errorf("error finding an open port: %v", err)
		}
		port = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	}

	fmt.Printf("Starting HTTP server on :%s\n", port)
	fmt.Printf("Access OpenAPI documentation at: http://localhost:%s/documentation/json\n", port)

	if filePath != "" {
		fmt.Printf("You can find the file at: http://localhost:%s/slang/%s\n", port, filePath)
	} else {
		fmt.Printf("Access the contract files at: http://localhost:%s/slang/\n", port)
	}

	// Start the HTTP server
	server := &http.Server{
		Handler:      mainRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("error starting HTTP server: %v", err)
	}

	return nil
}
