package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Example of using the list command in a folder with slang files.
func Example_listCmd() {

	// Prepare the command
	cmd := exec.Command("go", "run", "main.go", "list", "examples")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		fmt.Println("Command execution failed:", err)
		return
	}

	// Output the results
	output := out.String()
	fmt.Print(output)

	// Output:
	// Listing slangroom files in folder: examples
	// Found file: hello.slang (Path: examples/hello.slang)
	// Found file: test.slang (Path: examples/test.slang)
}

// Example of using the run command to execute a specific slangroom file.
func Example_runCmd() {

	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "run", "examples", "hello")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		fmt.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	// Running slangroom file: hello from folder: examples
	// {"output":["hello"]}
}
