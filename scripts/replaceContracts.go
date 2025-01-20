package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReplaceContracts temporarily replaces the contents of the contracts folder
// with the contents from a specified directory (e.g., for testing or custom build).
func replaceContracts(srcDir string) error {
	// Path to the original contracts folder
	contractsDir := "contracts"
	// Backup the original contracts folder
	backupDir := "contracts_backup"

	// Backup original contracts directory
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("failed to remove old backup: %w", err)
	}
	if err := os.Rename(contractsDir, backupDir); err != nil {
		return fmt.Errorf("failed to backup contracts folder: %w", err)
	}

	// Copy the new contents into the contracts directory
	fmt.Println(srcDir)
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	if err := os.MkdirAll(contractsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create contracts folder: %w", err)
	}

	for _, file := range files {
		srcFile := filepath.Join(srcDir, file.Name())
		destFile := filepath.Join(contractsDir, file.Name())

		// Copy file content
		input, err := os.ReadFile(srcFile)
		if err != nil {
			return fmt.Errorf("failed to read source file: %w", err)
		}
		if err := os.WriteFile(destFile, input, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write file to contracts: %w", err)
		}
	}

	return nil
}

func main() {
	srcDir := os.Getenv("CONTRACTS_DIR")
	if srcDir == "" {
		srcDir = "contracts" // default
	}
	if err := replaceContracts(srcDir); err != nil {
		fmt.Printf("Error replacing contracts: %v\n", err)
		os.Exit(1)
	}
}
