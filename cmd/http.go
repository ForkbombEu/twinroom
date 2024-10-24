package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/gorilla/mux"
)

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

func executeSlangFileHandler(file fouter.SlangFile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		result, err := slangroom.Exec(slangroom.SlangroomInput{Contract: file.Content})
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

func startHTTPServer(folder string, url string) error {
	r := mux.NewRouter()
	slangFiles := make(map[string][]string)

	err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
		relativePath := filepath.Join(file.Dir, strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName)))
		slangFiles[file.Dir] = append(slangFiles[file.Dir], relativePath)
		r.HandleFunc("/slang/"+relativePath, slangFilePageHandler(file)).Methods("GET")
		r.HandleFunc("/slang/execute/"+relativePath, executeSlangFileHandler(file)).Methods("POST")
	})

	if err != nil {
		return fmt.Errorf("error setting up file router: %v", err)
	}

	r.HandleFunc("/slang/", listSlangFilesHandler(slangFiles)).Methods("GET")

	fmt.Println("Starting HTTP server on :3000")
	if url != "" {
		fmt.Printf("You can find the file at: %s\n", url)
	} else {
		fmt.Println("Access the contract files at: http://localhost:3000/slang/")
	}

	if err := http.ListenAndServe(":3000", r); err != nil {
		return fmt.Errorf("error starting HTTP server: %v", err)
	}

	return nil
}

func GetSlangFileURL(folder string, fileName string) string {
	relativePath := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	slangFileURL := fmt.Sprintf("http://localhost:3000/slang/%s", relativePath)

	return slangFileURL
}
