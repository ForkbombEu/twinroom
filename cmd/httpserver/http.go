package httpserver

import (
	"context"
	"embed"
	"fmt"
	"net"
	"net/http"
	"time"
)

// define the input needed to start the server
type HTTPInput struct {
	BinaryName     string
	EmbeddedFolder *embed.FS
	EmbeddedPath   string
	EmbeddedSubDir string
	Path           string
	FileName       string
}

const openapiCSS = `
<style>
	.HttpOperation__Description h1:before {
		content: "#";
	}
	.HttpOperation__Description h1 {
		font-size: 16px;
		margin: 0 0;
		font-style: italic;
		color: #444;
		font-weight: 100;
	}
	.HttpOperation__Description p {
		white-space: pre-line;
	}
</style>
`

// StartSHTTPrver starts an HTTP server that serves the OpenAPI documentation via Stoplight Elements.
// The documentation is available at the `/slang` endpoint.
func StartHTTPServer(input HTTPInput) error {
	ctx := context.Background()

	// Generate OpenAPI router

	mainRouter, err := GenerateOpenAPIRouter(ctx, input)
	if err != nil {
		return fmt.Errorf("error generating OpenAPI router: %v", err)
	}
	// Define the handler for serving the Stoplight Elements HTML page
	mainRouter.HandleFunc("/slang", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		apiDescriptionURL := fmt.Sprintf("%s://%s/documentation/json", scheme, host)

		// Write the Stoplight Elements HTML page
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <title>%s</title>
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
	%s
  </head>
  <body>
    <elements-api layout="sidebar" router="hash" apiDescriptionUrl="%s" />
  </body>
</html>`, input.BinaryName, openapiCSS, apiDescriptionURL)
		if err != nil {
			fmt.Printf("Failed to write HTTP response: %v\n", err)
		}
	})

	// Set up a listener on port 3000 or an available port
	port := "3000"
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		listener, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return fmt.Errorf("error finding an open port: %v", err)
		}
		port = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
	}

	// Print server information
	fmt.Printf("Starting HTTP server on :%s\n", port)
	fmt.Printf("Access the API documentation at: http://localhost:%s/slang\n", port)

	// Start the HTTP server
	server := &http.Server{
		Handler:      mainRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error starting HTTP server: %v", err)
	}

	return nil
}
