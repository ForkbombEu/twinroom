package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/gemini/cmd/httpserver"
	"github.com/forkbombeu/gemini/cmd/utils"
	"github.com/spf13/cobra"
)

var contracts embed.FS
var daemon bool

var extension string = ".slang"

// rootCmd is the base command when called without any subcommands.
// It initializes the command tree and is responsible for starting the Gemini CLI.

func Execute(embeddedFiles embed.FS) {
	contracts = embeddedFiles

	// Dynamically add commands for each embedded file
	addEmbeddedFileCommands()

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
	runCmd.PersistentFlags().BoolVarP(&daemon, "daemon", "d", false, "Start HTTP server to execute slangroom file")
}

// listCmd is a command that lists all slangroom files in the folder or list embedded files if no folder is specified.
// It accepts an optional daemon flag to start an HTTP server for listing the files.
var listCmd = &cobra.Command{
	Use:   "list [folder]",
	Short: "List all contracts in the folder or list embedded contracts if no folder is specified",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// If no folder argument is provided, list embedded files
			fmt.Println("Listing embedded slangroom files:")

			// If the daemon flag is set, start the HTTP server
			if daemon {
				if err := httpserver.StartHTTPServer("contracts", "", nil); err != nil {
					fmt.Printf("Failed to start HTTP server: %v\n", err)
					os.Exit(1)
				}
				return
			}
			err := fouter.CreateFileRouter("", &contracts, "contracts", func(file fouter.SlangFile) {
				fmt.Printf("Found file: %s (Path: %s)\n", strings.TrimSuffix(file.FileName, extension), file.Path)
			})
			if err != nil {
				fmt.Println("Error:", err)
			}
		} else {
			// If a folder argument is provided, list files in that folder
			folder := args[0]
			fmt.Printf("Listing contracts in folder: %s\n", folder)

			if daemon {
				if err := httpserver.StartHTTPServer(folder, "", nil); err != nil {
					fmt.Printf("Failed to start HTTP server: %v\n", err)
					os.Exit(1)
				}
				return
			}

			// List slangroom files in the specified folder
			err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
				fmt.Printf("Found file: %s (Path: %s)\n", strings.TrimSuffix(file.FileName, extension), file.Path)
			})
			if err != nil {
				fmt.Println("Error:", err)
			}
		}
	},
}

// Function to add commands for each embedded slangroom file
func addEmbeddedFileCommands() {
	dirCommands := make(map[string]*cobra.Command)

	err := fouter.CreateFileRouter("", &contracts, "contracts", func(file fouter.SlangFile) {
		relativePath := strings.TrimPrefix(filepath.Join(file.Dir, file.FileName), "contracts/")
		relativePath = strings.TrimSuffix(relativePath, filepath.Ext(relativePath))

		pathParts := strings.Split(relativePath, string(os.PathSeparator))
		fileCmdName := pathParts[len(pathParts)-1]
		dirPath := strings.Join(pathParts[:len(pathParts)-1], string(os.PathSeparator))

		// Ensure a command exists for the directory path
		var parentCmd *cobra.Command = runCmd
		if dirPath != "" {
			if _, exists := dirCommands[dirPath]; !exists {
				dirCmd := &cobra.Command{
					Use:   strings.ReplaceAll(dirPath, string(os.PathSeparator), " "),
					Short: fmt.Sprintf("Commands for files in %s", dirPath),
				}
				dirCommands[dirPath] = dirCmd
				parentCmd.AddCommand(dirCmd)
			}
			parentCmd = dirCommands[dirPath]
		}

		// Create the command for the file
		fileCmd := &cobra.Command{
			Use:   fileCmdName,
			Short: fmt.Sprintf("Execute the embedded contract %s", file.FileName),
		}
		var isMetadata bool
		argContents := make(map[string]string)
		flagContents := make(map[string]utils.FlagData)

		metadataPath := filepath.Join(file.Dir, strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))+".metadata.json")
		metadata, err := utils.LoadMetadata(&contracts, metadataPath)
		if err == nil {
			isMetadata = true
			// Set command description
			fileCmd.Short = metadata.Description
			argContents, flagContents = utils.ConfigureArgumentsAndFlags(fileCmd, metadata)
			fileCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
				return utils.ValidateFlags(cmd, flagContents, argContents)
			}

		}

		// Set the command's run function
		fileCmd.Run = func(cmd *cobra.Command, args []string) {
			runFileCommand(file, args, metadata, argContents, isMetadata, relativePath)
		}

		// Add the file command to its directory's command
		parentCmd.AddCommand(fileCmd)
	})

	if err != nil {
		fmt.Println("Error adding embedded file commands:", err)
	}
}

// runCmd is a command that executes a specific slangroom file from a given folder.
// It accepts a folder and file path and can optionally start an HTTP server if the daemon flag is set.
var runCmd = &cobra.Command{
	Use:   "gemini [folder]",
	Short: "Execute a specific slangroom file in a dynamically specified folder",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		folder := args[0]
		filePath := filepath.Join(args[1:]...)
		input := slangroom.SlangroomInput{}
		found := false

		// List and execute slangroom files in the specified folder
		err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
			relativeFilePath := filepath.Join(file.Dir, file.FileName)
			relativeFilePath = strings.TrimSuffix(relativeFilePath, filepath.Ext(relativeFilePath))
			filename := strings.TrimSuffix(file.FileName, extension)

			if relativeFilePath == filePath {
				found = true
				input.Contract = file.Content
				utils.LoadAdditionalData(filepath.Join(folder, file.Dir), filename, &input)

				// Start HTTP server if daemon flag is set
				if daemon {
					if err := httpserver.StartHTTPServer(folder, filePath, nil); err != nil {
						fmt.Printf("Failed to start HTTP server: %v\n", err)
						os.Exit(1)
					}
					return
				}

				// Execute the slangroom file
				res, err := slangroom.Exec(input)
				if err != nil {
					fmt.Println("Error:", err)
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

		if !found {
			fmt.Printf("File %s not found in folder %s\n", filePath, folder)
		}
	},
}

func runFileCommand(file fouter.SlangFile, args []string, metadata *utils.CommandMetadata, argContents map[string]string, isMetadata bool, relativePath string) {
	input := slangroom.SlangroomInput{Contract: file.Content}
	filename := strings.TrimSuffix(file.FileName, extension)
	utils.LoadAdditionalData(file.Dir, filename, &input)
	if isMetadata {
		for i, arg := range args {
			if i < len(metadata.Arguments) {
				argContents[utils.NormalizeArgumentName(metadata.Arguments[i].Name)] = arg
			}
		}
		// Convert argContents to JSON if needed
		jsonData, err := json.Marshal(argContents)
		if err != nil {
			fmt.Println("Error encoding arguments to JSON:", err)
			return
		}
		if input.Data != "" {
			if input.Data, err = utils.MergeJSON(input.Data, string(jsonData)); err != nil {
				fmt.Println("Error encoding arguments to JSON:", err)
				os.Exit(1)
			}
		} else {
			input.Data = string(jsonData)
		}
	}
	// Start HTTP server if daemon flag is set
	if daemon {
		if err := httpserver.StartHTTPServer("contracts", relativePath, &input); err != nil {
			fmt.Printf("Failed to start HTTP server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Execute the slangroom file
	res, err := slangroom.Exec(input)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println(res.Logs)
	} else {
		fmt.Println(res.Output)
	}
}
