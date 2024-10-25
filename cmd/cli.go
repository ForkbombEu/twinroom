package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/gemini/cmd/httpserver"
	"github.com/spf13/cobra"
)

var contracts embed.FS
var daemon bool

// rootCmd is the base command when called without any subcommands.
// It initializes the command tree and is responsible for starting the Gemini CLI.

func Execute(embeddedFiles embed.FS) {

	contracts = embeddedFiles

	// Execute the root command
	if err := runCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	runCmd.AddCommand(listCmd)
	// Add a flag for the daemon mode to the 'list' command
	listCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to list slangroom files")
	runCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to execute slangroom file")
}

// listCmd is a command that lists all slangroom files in the folder or list embedded files if no folder is specified.
// It accepts an optional daemon flag to start an HTTP server for listing the files.
var listCmd = &cobra.Command{
	Use:   "list [folder]",
	Short: "List all slangroom files in the folder or list embedded files if no folder is specified",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// If no folder argument is provided, list embedded files
			fmt.Println("Listing embedded slangroom files:")

			// If the daemon flag is set, start the HTTP server
			if daemon {
				if err := httpserver.StartHTTPServer("contracts", ""); err != nil {
					fmt.Printf("Failed to start HTTP server: %v\n", err)
					os.Exit(1)
				}
				return
			}
			err := fouter.CreateFileRouter("", &contracts, "contracts", func(file fouter.SlangFile) {
				fmt.Printf("Found file: %s (Path: %s)\n", file.FileName, file.Path)
			})
			if err != nil {
				fmt.Println("Error:", err)
			}
		} else {
			// If a folder argument is provided, list files in that folder
			folder := args[0]
			fmt.Printf("Listing slangroom files in folder: %s\n", folder)

			if daemon {
				if err := httpserver.StartHTTPServer(folder, ""); err != nil {
					fmt.Printf("Failed to start HTTP server: %v\n", err)
					os.Exit(1)
				}
				return
			}

			// List slangroom files in the specified folder
			err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
				fmt.Printf("Found file: %s (Path: %s)\n", file.FileName, file.Path)
			})
			if err != nil {
				fmt.Println("Error:", err)
			}
		}
	},
}

// runCmd is a command that executes a specific slangroom file from a given folder.
// It accepts a folder and file path and can optionally start an HTTP server if the daemon flag is set.
var runCmd = &cobra.Command{
	Use:   "gemini [folder or file]",
	Short: "Execute a specific slangroom file",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := filepath.Join(args...)

		found := false

		// Check if filePath exists in embedded files
		err := fouter.CreateFileRouter("", &contracts, "contracts", func(file fouter.SlangFile) {
			relativePath := strings.TrimPrefix(filepath.Join(file.Dir, file.FileName), "contracts/")
			relativePath = strings.TrimSuffix(relativePath, filepath.Ext(relativePath))

			if relativePath == filePath {
				found = true
				input := slangroom.SlangroomInput{Contract: file.Content}

				// If daemon flag is set, start HTTP server for the embedded file
				if daemon {
					fileURL := httpserver.GetSlangFileURL("contracts", filePath)
					if err := httpserver.StartHTTPServer("contracts", fileURL); err != nil {
						fmt.Printf("Failed to start HTTP server: %v\n", err)
						os.Exit(1)
					}
					return
				}

				res, err := slangroom.Exec(input)
				if err != nil {
					fmt.Println(res.Logs)
				} else {
					fmt.Println(res.Output)
				}
			}
		})

		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// If not found in embedded, check on the filesystem
		if !found && len(args) >= 2 {
			folder := args[0]
			filePath := filepath.Join(args[1:]...)

			err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
				relativeFilePath := filepath.Join(file.Dir, file.FileName)
				relativeFilePath = strings.TrimSuffix(relativeFilePath, filepath.Ext(relativeFilePath))

				if relativeFilePath == filePath {
					found = true
					input := slangroom.SlangroomInput{Contract: file.Content}

					if daemon {
						fileURL := httpserver.GetSlangFileURL(folder, filePath)
						if err := httpserver.StartHTTPServer(folder, fileURL); err != nil {
							fmt.Printf("Failed to start HTTP server: %v\n", err)
							os.Exit(1)
						}
						return
					}

					res, err := slangroom.Exec(input)
					if err != nil {
						fmt.Println(res.Logs)
					} else {
						fmt.Println(res.Output)
					}
				}
			})

			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// If the file was not found in the provided folder path
			if !found {
				fmt.Printf("File %s not found in %s or embedded directory\n", filePath, folder)
			}
		} else if !found {
			fmt.Printf("File %s not found in embedded directory\n", filePath)
		}
	},
}
