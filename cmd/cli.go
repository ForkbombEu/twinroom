package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForkbombEu/fouter"
	slangroom "github.com/dyne/slangroom-exec/bindings/go"
	"github.com/forkbombeu/twinroom/cmd/httpserver"
	"github.com/forkbombeu/twinroom/cmd/utils"
	"github.com/spf13/cobra"
)

var contracts embed.FS
var daemon bool

// runCmd is the base command when called without any subcommands.
func Execute(embeddedFiles embed.FS) {
	contracts = embeddedFiles

	// Dynamically add commands for each embedded file
	addEmbeddedFileCommands()

	// Execute the root command
	if err := runCmd.Execute(); err != nil {
		log.Println(err)
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
	Run: func(_ *cobra.Command, args []string) {
		if len(args) == 0 {
			// If no folder argument is provided, list embedded files
			fmt.Println("Listing embedded slangroom files:")
			err := fouter.CreateFileRouter("", &contracts, "contracts", func(file fouter.SlangFile) {
				relativePath := strings.TrimPrefix(filepath.Join(file.Dir, file.FileName), "contracts/")
				relativePath = strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
				fmt.Printf("Found file: %s\n", relativePath)
			})
			if err != nil {
				log.Println("Error:", err)
			}
		} else {
			// If a folder argument is provided, list files in that folder
			folder := args[0]
			fmt.Printf("Listing contracts in folder: %s\n", folder)
			err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
				relativeFilePath := filepath.Join(file.Dir, file.FileName)
				relativeFilePath = strings.TrimSuffix(relativeFilePath, filepath.Ext(relativeFilePath))
				fmt.Printf("Found file: %s\n", relativeFilePath)
			})
			if err != nil {
				log.Println("Error:", err)
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
				dirCmd.Run = func(_ *cobra.Command, _ []string) {
					if daemon {
						httpInput := httpserver.HTTPInput{
							BinaryName:     filepath.Base(os.Args[0]),
							EmbeddedFolder: &contracts,
							EmbeddedPath:   "contracts",
							EmbeddedSubDir: dirPath,
						}
						if err := httpserver.StartHTTPServer(httpInput); err != nil {
							log.Printf("Failed to start HTTP server: %v\n", err)
							os.Exit(1)
						}
						return
					}
				}
				dirCommands[dirPath] = dirCmd
				parentCmd.AddCommand(dirCmd)
			}
			parentCmd = dirCommands[dirPath]
		}

		// Create the command for the file
		fileCmd := &cobra.Command{
			Use:   fileCmdName,
			Short: fmt.Sprintf("Execute the embedded contract %s", strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))),
		}
		var isMetadata bool
		argContents := make(map[string]interface{})
		flagContents := make(map[string]utils.FlagData)

		input := slangroom.SlangroomInput{Contract: file.Content}

		metadataPath := filepath.Join(file.Dir, strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))+".metadata.json")
		metadata, err := utils.LoadMetadata(&contracts, metadataPath)
		if err != nil && err.Error() != "metadata file not found" {
			log.Printf("WARNING: error in metadata for contracts: %s\n", fileCmdName)
			log.Println(err)
		} else if err == nil {
			isMetadata = true
			// Set command description
			fileCmd.Short = metadata.Description
			argContents, flagContents, err = utils.ConfigureArgumentsAndFlags(fileCmd, metadata, "")
			if err != nil {
				log.Printf("Failed to set arguments or flags: %v\n", err)
				os.Exit(1)
			}
			fileCmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
				return utils.ValidateFlags(cmd, flagContents, argContents, &input)
			}
		} else {
			introspectionData, err := slangroom.Introspect(file.Content)
			if err == nil {
				introspectionData = utils.CleanIntrospection(file.Content, introspectionData)
			} else {
				introspectionData = ""
			}
			argContents, flagContents, err = utils.ConfigureArgumentsAndFlags(fileCmd, metadata, introspectionData)
			if err != nil {
				log.Printf("Failed to set arguments or flags: %v\n", err)
				os.Exit(1)
			}
			if introspectionData != "" && introspectionData != "{}" {
				isMetadata = true
			}
		}

		fileCmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
			return utils.ValidateFlags(cmd, flagContents, argContents, &input)
		}
		// Set the command's run function
		fileCmd.Run = func(_ *cobra.Command, args []string) {
			runFileCommand(file, args, metadata, argContents, isMetadata, &input)
		}

		// Add the file command to its directory's command
		parentCmd.AddCommand(fileCmd)
	})

	if err != nil {
		log.Println("Error adding embedded file commands:", err)
	}
}

// runCmd is a command that executes a specific slangroom file from a given folder.
// It accepts a folder and file path and can optionally start an HTTP server if the daemon flag is set.
var runCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]) + " [folder]",
	Short: "Execute a specific slangroom file in a dynamically specified folder or in the embedded folder contracts",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if daemon {
			if len(args) == 0 {
				httpInput := httpserver.HTTPInput{
					BinaryName:     filepath.Base(os.Args[0]),
					EmbeddedFolder: &contracts,
					EmbeddedPath:   "contracts",
				}
				if err := httpserver.StartHTTPServer(httpInput); err != nil {
					log.Printf("Failed to start HTTP server: %v\n", err)
					os.Exit(1)
				}
				return
			}
			folder := args[0]
			filePath := filepath.Join(args[1:]...)
			isDir, err := utils.IsDir(filepath.Join(folder, filePath))
			if isDir && err == nil {
				httpInput := httpserver.HTTPInput{
					BinaryName: filepath.Base(os.Args[0]),
					Path:       filepath.Join(folder, filePath),
				}
				if err := httpserver.StartHTTPServer(httpInput); err != nil {
					log.Printf("Failed to start HTTP server: %v\n", err)
					os.Exit(1)
				}
				return
			}
		}

		if len(args) < 1 {
			log.Println("Error: folder or  argument is required")
			if err := cmd.Help(); err != nil {
				log.Printf("Failed to start the program: %v\n", err)
				os.Exit(1)
			}

			return
		}
		folder := args[0]
		filePath := filepath.Join(args[1:]...)
		input := slangroom.SlangroomInput{}
		found := false

		// List and execute slangroom files in the specified folder
		err := fouter.CreateFileRouter(folder, nil, "", func(file fouter.SlangFile) {
			relativeFilePath := filepath.Join(file.Dir, file.FileName)
			relativeFilePath = strings.TrimSuffix(relativeFilePath, filepath.Ext(relativeFilePath))
			filename := strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))

			if relativeFilePath == filePath {
				found = true
				input.Contract = file.Content
				err := utils.LoadAdditionalData(filepath.Join(folder, file.Dir), filename, &input)

				if err != nil {
					log.Printf("Failed to load data from JSON file: %v\n", err)
					os.Exit(1)
				}

				// Start HTTP server if daemon flag is set
				if daemon {
					httpInput := httpserver.HTTPInput{
						BinaryName: filepath.Base(os.Args[0]),
						Path:       file.Path,
					}
					if err := httpserver.StartHTTPServer(httpInput); err != nil {
						log.Printf("Failed to start HTTP server: %v\n", err)
						os.Exit(1)
					}
					return
				}

				// Execute the slangroom file
				res, err := slangroom.Exec(input)
				if err != nil {
					log.Println("Error:", err)
					log.Println(res.Logs)
				} else {
					fmt.Println(res.Output)
				}
			}
		})

		if err != nil {
			log.Println("Error:", err)
			return
		}

		if !found {
			log.Printf("File %s not found in folder %s\n", filePath, folder)
		}
	},
}

func runFileCommand(file fouter.SlangFile, args []string, metadata *utils.CommandMetadata, argContents map[string]interface{}, isMetadata bool, input *slangroom.SlangroomInput) {
	filename := strings.TrimSuffix(file.FileName, filepath.Ext(file.FileName))
	err := utils.LoadAdditionalData(file.Dir, filename, input)
	if err != nil {
		log.Printf("Failed to load data from JSON file: %v\n", err)
		os.Exit(1)
	}
	if isMetadata {
		for key, value := range metadata.Environment {
			if err := os.Setenv(key, value); err != nil {
				log.Println("Failed to set environment variable:", key)
				os.Exit(1)
			}
		}
		for i, arg := range args {
			if i < len(metadata.Arguments) {
				argContents[utils.NormalizeArgumentName(metadata.Arguments[i].Name)] = arg
			}
		}
		// Convert argContents to JSON if needed
		jsonData, err := json.Marshal(argContents)
		if err != nil {
			log.Println("Error encoding arguments to JSON:", err)
			return
		}
		if input.Data != "" {
			if input.Data, err = utils.MergeJSON(input.Data, string(jsonData)); err != nil {
				log.Println("Error encoding arguments to JSON:", err)
				os.Exit(1)
			}
		} else {
			input.Data = string(jsonData)
		}
	}
	// Start HTTP server if daemon flag is set
	if daemon {
		httpInput := httpserver.HTTPInput{
			BinaryName:     filepath.Base(os.Args[0]),
			EmbeddedFolder: &contracts,
			EmbeddedPath:   "contracts",
			FileName:       filename,
		}
		if err := httpserver.StartHTTPServer(httpInput); err != nil {
			log.Printf("Failed to start HTTP server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Execute the slangroom file
	res, err := slangroom.Exec(*input)
	if err != nil {
		log.Println("Error:", err)
		log.Println(res.Logs)
	} else {
		fmt.Println(res.Output)
	}
}
