package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
)

func TestListSlangFilesHandler(t *testing.T) {
	slangFiles := map[string][]string{
		"example_dir": {"file1.slang", "file2.slang"},
	}

	req, err := http.NewRequest("GET", "/slang/", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := listSlangFilesHandler(slangFiles)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !contains(body, "Available contract files:") {
		t.Errorf("Expected response to contain 'Available contract files:', got %v", body)
	}
	if !contains(body, "file1.slang") || !contains(body, "file2.slang") {
		t.Errorf("Expected response to contain 'file1.slang' and 'file2.slang', got %v", body)
	}
}

func TestSlangFilePageHandler(t *testing.T) {
	file := fouter.SlangFile{
		FileName: "test.slang",
		Content:  "contract content",
		Dir:      "example_dir",
		Path:     "example_dir/test.slang",
	}

	req, err := http.NewRequest("GET", "/slang/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := slangFilePageHandler(file)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !contains(body, "contract content") {
		t.Errorf("Expected response to contain 'contract content', got %v", body)
	}
	if !contains(body, "Execute test.slang") {
		t.Errorf("Expected response to contain 'Execute test.slang', got %v", body)
	}
}

func TestExecuteSlangFileHandler_Success(t *testing.T) {
	file := fouter.SlangFile{
		FileName: "test.slang",
		Content: `Given nothing
Then print the string 'Hello'`,
	}

	req, err := http.NewRequest("POST", "/slang/execute/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := executeSlangFileHandler(file, "", nil)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["output"] != `{"output":["Hello"]}` {
		t.Errorf("Expected output to be 'Hello', got %v", response["output"])
	}
}
func TestExecuteSlangFileHandler_With_Data(t *testing.T) {
	file := fouter.SlangFile{
		FileName: "test.slang",
		Content: `Given nothing
Then print the string 'Hello'`,
	}
	input := slangroom.SlangroomInput{Contract: file.Content}
	req, err := http.NewRequest("POST", "/slang/execute/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := executeSlangFileHandler(file, "", &input)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["output"] != `{"output":["Hello"]}` {
		t.Errorf("Expected output to be 'Hello', got %v", response["output"])
	}
}

func TestExecuteSlangFileHandler_MethodNotAllowed(t *testing.T) {
	file := fouter.SlangFile{
		FileName: "test.slang",
		Content:  "contract content",
	}

	req, err := http.NewRequest("GET", "/slang/execute/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := executeSlangFileHandler(file, "", nil)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %v, got %v", http.StatusMethodNotAllowed, rr.Code)
	}

	body := rr.Body.String()
	if !contains(body, "Invalid request method") {
		t.Errorf("Expected response to contain 'Invalid request method', got %v", body)
	}
}

// Helper function to check if a substring is in a string
func contains(str, substr string) bool {
	return len(str) >= len(substr) && strings.Contains(str, substr)
}
