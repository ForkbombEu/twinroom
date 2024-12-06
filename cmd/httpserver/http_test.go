package httpserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	swagger "github.com/davidebianchi/gswagger"
	"github.com/stretchr/testify/require"
)

func TestGenerateOpenAPIRouter(t *testing.T) {
	// Mock input for HTTPInput
	input := HTTPInput{
		BinaryName:     "TestBinary",
		Path:           "testdata/slang", // Mocked slang directory for testing
		EmbeddedFolder: nil,
		EmbeddedPath:   "",
	}

	// Create mock slang file in testdata directory
	mockSlangFile := "testdata/slang/example.slang"
	mockSlangContent := `Rule unknown ignore
Given I have a 'string' named 'test'
Then print the data
`
	err := os.MkdirAll(filepath.Dir(mockSlangFile), os.ModePerm)
	require.NoError(t, err)
	err = os.WriteFile(mockSlangFile, []byte(mockSlangContent), 0600)
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll("./testdata"); err != nil {
			t.Errorf("Failed to remove test directory: %v", err)
		}
	}()

	ctx := context.Background()

	// Generate router
	muxRouter, err := GenerateOpenAPIRouter(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, muxRouter)

	t.Run("correctly exposes OpenAPI documentation", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, swagger.DefaultJSONDocumentationPath, nil)

		muxRouter.ServeHTTP(w, req)
		body := w.Result().Body // //nolint:bodyclose
		defer func() {
			if err := body.Close(); err != nil {
				t.Errorf("Failed to close body: %v", err)
			}
		}()

		// These lines do not require closing since they just access StatusCode and Headers
		require.Equal(t, http.StatusOK, w.Result().StatusCode)                      //nolint:bodyclose
		require.Equal(t, "application/json", w.Result().Header.Get("Content-Type")) //nolint:bodyclose

		bodycontent, err := io.ReadAll(body)
		require.NoError(t, err)
		fmt.Println(string(bodycontent))
		expected := `{"info":{"title":"TestBinary","version":"1.0.0"},"openapi":"3.0.0","paths":{"/example":{"get":{"description":"Rule unknown ignore\n\nGiven I have a 'string' named 'test'\n\nThen print the data\n\n","parameters":[{"description":"The test","in":"query","name":"test","schema":{"type":"string"}}],"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"output":{"items":{"type":"string"},"type":"array"}},"required":["output"],"type":"object"}}},"description":"The slangroom execution output, splitted by newline"},"500":{"content":{"application/json":{"schema":{"properties":{"message":{"items":{"type":"string"},"type":"array"}},"required":["message"],"type":"object"}}},"description":"Slangroom execution error"}},"tags":["📑 Zencodes"]},"post":{"description":"Rule unknown ignore\n\nGiven I have a 'string' named 'test'\n\nThen print the data\n\n","requestBody":{"content":{"application/json":{"schema":{"properties":{"test":{"type":"string"}},"required":["test"],"type":"object"}}}},"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"output":{"items":{"type":"string"},"type":"array"}},"required":["output"],"type":"object"}}},"description":"The slangroom execution output, split by newline"},"500":{"content":{"application/json":{"schema":{"properties":{"message":{"items":{"type":"string"},"type":"array"}},"required":["message"],"type":"object"}}},"description":"Slangroom execution error"}},"tags":["📑 Zencodes"]}}},"tags":[{"description":"Endpoints generated over the Zencode smart contracts","name":"📑 Zencodes"}]}`
		require.JSONEq(t, expected, string(bodycontent), "actual json data: %s", body)
	})

	// Test POST route execution
	t.Run("correctly handles POST route", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/example", nil) // Mocked route

		muxRouter.ServeHTTP(w, req)
		body := w.Result().Body //nolint:bodyclose
		defer func() {
			if err := body.Close(); err != nil {
				t.Errorf("Failed to close body: %v", err)
			}
		}()

		require.Equal(t, http.StatusOK, w.Result().StatusCode) //nolint:bodyclose
		require.Equal(t, "OK", w.Body.String())
	})

	// Test GET route execution
	t.Run("correctly handles GET route", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/example", nil) // Mocked route

		muxRouter.ServeHTTP(w, req)
		body := w.Result().Body //nolint:bodyclose
		defer func() {
			if err := body.Close(); err != nil {
				t.Errorf("Failed to close body: %v", err)
			}
		}()

		require.Equal(t, http.StatusOK, w.Result().StatusCode) //nolint:bodyclose
	})
}
