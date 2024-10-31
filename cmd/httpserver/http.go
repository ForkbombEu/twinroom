package httpserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/gemini/cmd/utils"
	"github.com/gorilla/mux"
)

var port string

// listSlangFilesHandler returns an HTTP handler that lists available slangroom files in the provided directories.
// It generates an HTML page displaying the slangroom files for each directory.
func listSlangFilesHandler(slangFiles map[string][]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "<html><body><h1>Available contract files:</h1>")

		for dir, files := range slangFiles {
			fmt.Fprintf(w, "<h2>Directory: %s</h2><ul>", dir)

			for _, file := range files {
				fileName := filepath.Base(file)
				link := fmt.Sprintf("/slang/%s", file)
				fmt.Fprintf(w, `<li><a href="%s">%s/%s</a></li>`, link, dir, fileName)
			}

			fmt.Fprintln(w, "</ul>")
		}

		fmt.Fprintln(w, "</body></html>")
	}
}

// slangFilePageHandler returns an HTTP handler that displays the contents of a single slangroom file.
// It generates an HTML page showing the content of the file along with a button to execute it.
func slangFilePageHandler(file fouter.SlangFile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><h1>%s</h1><pre>%s</pre>", name, file.Content)
		relativePath := filepath.Join(file.Dir, name)
		fmt.Fprintf(w, `<form method="POST" action="/slang/execute/%s">
                            <button type="submit">Execute %s</button>
                        </form>`, relativePath, filepath.Base(file.Path))

		fmt.Fprintln(w, "</body></html>")
	}
}

// executeSlangFileHandler returns an HTTP handler that executes a slangroom file via a POST request.
// The handler responds with a JSON output of the result or an error if the execution fails.
func executeSlangFileHandler(file fouter.SlangFile, baseFolder string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		input := slangroom.SlangroomInput{Contract: file.Content}

		// Load additional data from JSON files with matching names
		filename := strings.TrimSuffix(file.FileName, ".slang")
		err := utils.LoadAdditionalData(filepath.Join(baseFolder, file.Dir), filename, &input)
		if err != nil {
			http.Error(w, "Error loading additional data: "+err.Error(), http.StatusInternalServerError)
			return
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
func StartHTTPServer(folder string, filePath string) error {
	r := mux.NewRouter()
	slangFiles := make(map[string][]string)

	err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
		relativePath := filepath.Join(file.Dir, strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName)))
		slangFiles[file.Dir] = append(slangFiles[file.Dir], relativePath)
		r.HandleFunc("/slang/"+relativePath, slangFilePageHandler(file)).Methods("GET")
		r.HandleFunc("/slang/execute/"+relativePath, executeSlangFileHandler(file, folder)).Methods("POST")
	})

	if err != nil {
		return fmt.Errorf("error setting up file router: %v", err)
	}

	r.HandleFunc("/slang/", listSlangFilesHandler(slangFiles)).Methods("GET")

	// Try binding to port 3000, otherwise bind to a random open port
	port = "3000"
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			return fmt.Errorf("error finding an open port: %v", err)
		}
		port = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	}

	fmt.Printf("Starting HTTP server on :%s\n", port)
	if filePath != "" {
		fmt.Printf("You can find the file at: http://localhost:%s/slang/%s", port, filePath)
	} else {
		fmt.Printf("Access the contract files at: http://localhost:%s/slang/\n", port)
	}

	if err := http.Serve(listener, r); err != nil {
		return fmt.Errorf("error starting HTTP server: %v", err)
	}

	return nil
}
