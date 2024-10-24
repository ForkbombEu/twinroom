package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/gemini/cmd/httpserver"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Slangroom double sided executor",
	Long:  "Gemini reads and executes slangroom contracts.",
}

// rootCmd is the base command when called without any subcommands.
// It initializes the command tree and is responsible for starting the Gemini CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add the 'list' and 'run' subcommands to the root command.
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
}

// listCmd is a command that lists all the slangroom files in a specified folder recursively.
// It accepts an optional daemon flag to start an HTTP server for listing the files.
var listCmd = &cobra.Command{
	Use:   "list [folder]",
	Short: "List all slangroom files in the folder recursively",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		folder := args[0]
		fmt.Printf("Listing slangroom files in folder: %s\n", folder)

		// If the daemon flag is set, start the HTTP server
		if daemon {
			if err := httpserver.StartHTTPServer(folder, ""); err != nil {
				fmt.Printf("Failed to start HTTP server: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Otherwise, list the slangroom files in the folder
		err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
			fmt.Printf("Found file: %s (Path: %s)\n", file.FileName, file.Path)
		})
		if err != nil {
			fmt.Println("Error:", err)
		}
	},
}

var daemon bool

func init() {
	// Add a flag for the daemon mode to the 'list' command.
	listCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to list slangroom files")
}

// runCmd is a command that executes a specific slangroom file from a given folder.
// It accepts a folder and file path and can optionally start an HTTP server if the daemon flag is set.
var runCmd = &cobra.Command{
	Use:   "run [folder] [file]",
	Short: "Execute a specific slangroom file",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		folder := args[0]
		filePath := filepath.Join(args[1:]...)

		// If the daemon flag is set, start the HTTP server
		if daemon {
			fileURL := httpserver.GetSlangFileURL(folder, filePath)
			if err := httpserver.StartHTTPServer(folder, fileURL); err != nil {
				fmt.Printf("Failed to start HTTP server: %v\n", err)
				os.Exit(1) // Exit the CLI with error status
			}

		} else {
			fmt.Printf("Running slangroom file: %s from folder: %s\n", filePath, folder)

			found := false
			err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
				relativeFilePath := filepath.Join(file.Dir, file.FileName)
				relativeFilePath = strings.TrimSuffix(relativeFilePath, filepath.Ext(relativeFilePath))
				if relativeFilePath == filePath {
					found = true
					input := slangroom.SlangroomInput{Contract: file.Content}
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
			}

			// If the file was not found in the folder, print an error message
			if !found {
				fmt.Printf("File %s not found in %s\n", filePath, folder)
			}
		}
	},
}

func init() {
	// Add a flag for the daemon mode to the 'run' command.
	runCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to execute slangroom file")
}
