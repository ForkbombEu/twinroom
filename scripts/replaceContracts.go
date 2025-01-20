package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ContractsConfig represents the structure of the JSON file that contains the paths.
type ContractsConfig struct {
	Paths []string `json:"paths"`
}

// ReplaceContracts temporarily replaces the contents of the contracts folder
// with the contents from the specified directories.
func replaceContracts(config ContractsConfig) error {
	// Path to the original contracts folder
	contractsDir := "contracts"
	// Backup the original contracts folder
	backupDir := "contracts_backup"

	// Check if there are any paths specified in the config
	if len(config.Paths) == 0 {
		return nil
	}

	// Backup original contracts directory
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("failed to remove old backup: %w", err)
	}
	if err := os.Rename(contractsDir, backupDir); err != nil {
		return fmt.Errorf("failed to backup contracts folder: %w", err)
	}

	// Copy the new contents into the contracts directory
	for _, srcDir := range config.Paths {
		files, err := os.ReadDir(srcDir)
		if err != nil {
			return fmt.Errorf("failed to read source directory %s: %w", srcDir, err)
		}

		if err := os.MkdirAll(contractsDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create contracts folder: %w", err)
		}

		// Copy files from each source directory into the contracts directory
		for _, file := range files {
			srcFile := filepath.Join(srcDir, file.Name())
			destFile := filepath.Join(contractsDir, file.Name())

			input, err := os.ReadFile(srcFile)
			if err != nil {
				return fmt.Errorf("failed to read source file %s: %w", srcFile, err)
			}
			if err := os.WriteFile(destFile, input, 0600); err != nil {
				return fmt.Errorf("failed to write file to contracts: %w", err)
			}
		}
	}

	return nil
}

func main() {
	// Read the config file with paths
	configFile := "extra_dir.json" // The JSON file with paths
	file, err := os.Open(configFile)
	if err != nil {
		fmt.Fprint(os.Stderr, "Warning: extra_dir.json not found, use default contracts folder.\n")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %v\n", err)
		}
	}()

	// Parse the JSON file
	var config ContractsConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		fmt.Fprint(os.Stderr, "Warning: extra_dir.json format not correct, use default contract folder.\n")
		return
	}

	// Replace contracts with the files from the listed directories
	if err := replaceContracts(config); err != nil {
		fmt.Printf("Error replacing contracts: %v\n", err)
		os.Exit(1)
	}
}
