package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Slangroom double sided executor",
	Long:  "Gemini reads and executes slangroom contracts.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
}

var listCmd = &cobra.Command{
	Use:   "list [folder]",
	Short: "List all slangroom files in the folder recursively",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		folder := args[0]
		fmt.Printf("Listing slangroom files in folder: %s\n", folder)

		if daemon {
			if err := startHTTPServer(folder, ""); err != nil {
				fmt.Printf("Failed to start HTTP server: %v\n", err)
				os.Exit(1)
			}
			return
		}

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
	listCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to list slangroom files")
}

var runCmd = &cobra.Command{
	Use:   "run [folder] [file]",
	Short: "Execute a specific slangroom file",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		folder := args[0]
		filePath := filepath.Join(args[1:]...)

		if daemon {
			fileURL := GetSlangFileURL(folder, filePath)
			if err := startHTTPServer(folder, fileURL); err != nil {
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

			if !found {
				fmt.Printf("File %s not found in %s\n", filePath, folder)
			}
		}
	},
}

func init() {
	runCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to execute slangroom file")
}
