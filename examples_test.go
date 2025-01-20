package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// Example of using the list command in a folder with slang files.
func Example_listCmd() {
	// Prepare the command
	cmd := exec.Command("go", "run", "main.go", "list", "contracts")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	output := out.String()
	fmt.Print(output)

	// Output:
	// Listing contracts in folder: contracts
	// Found file: test/broken
	// Found file: test/env
	// Found file: test/execute_zencode
	// Found file: test/hello
	// Found file: test/introspection
	// Found file: test/param
	// Found file: test/read_from_path
	// Found file: test/stdin
	// Found file: test/test
	// Found file: test/withschema

}

// Example of using the run command to execute a specific slangroom file.
func Example_runCmd() {
	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "test", "hello")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	// {"output":["Hello_from_embedded!"]}
}
func Example_runCmdWithExtraData() {
	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "test", "execute_zencode")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	// {"result":{"ecdh_public_key":"BLOYXryyAI7rPuyNbI0/1CfLFd7H/NbX+osqyQHjPR9BPK1lYSPOixZQWvFK+rkkJ+aJbYp6kii2Y3+fZ5vl2MA="}}
}

func Example_runCmdWithParam() {
	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "test", "param", "username", "-n", "testname", "-D", "small")

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	// {"drink":"small","name":"testname","timeout":60,"username":"username"}
}

func Example_runCmdWithEnvVariable() {
	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "test", "env")
	err := os.Setenv("FILE_CONTENT", "The enviroment variable is set correctly")
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}
	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	//{"content":"The enviroment variable is set correctly"}
}

func Example_runCmdWithStdinInput() {
	// Prepare the command to run the slang file
	cmd1 := exec.Command("cat", "contracts/test/hello.txt")
	cmd2 := exec.Command("go", "run", "main.go", "test", "stdin", "-f", "-")
	pipe, err := cmd1.StdoutPipe()
	if err != nil {
		log.Println("Command execution failed:", err)
	}
	var out bytes.Buffer
	cmd2.Stdin = pipe
	cmd2.Stdout = &out
	// Step 4: Start cmd1 and cmd2
	if err := cmd1.Start(); err != nil {
		log.Println("Command execution failed:", err)
	}

	if err := cmd2.Start(); err != nil {
		log.Println("Command execution failed:", err)
	}

	// Step 5: Wait for both commands to finish
	if err := cmd1.Wait(); err != nil {
		log.Println("Command execution failed:", err)
	}

	if err := cmd2.Wait(); err != nil {
		log.Println("Command execution failed:", err)
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	//{"file":"Hello World!"}
}

func Example_runCmdWithFilePath() {
	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "test", "stdin", "-f", "contracts/test/hello.txt")
	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	//{"file":"Hello World!"}
}

func Example_runCmdNeedEnvVariable() {
	// Prepare the command to run the slang file
	cmd := exec.Command("go", "run", "main.go", "test", "read_from_path", "-p", "hello.txt")
	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	// Check for errors
	if err != nil {
		log.Println("Command execution failed:", err)
		return
	}

	// Output the results
	fmt.Print(out.String())

	// Output:
	//{"file_content":"Hello World!\n"}
}
