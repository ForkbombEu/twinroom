package main

import (
	"embed"

	"github.com/forkbombeu/twinroom/cmd"
)

//go:embed contracts
var contracts embed.FS

func main() {
	// Initialize CLI with embedded contracts
	cmd.Execute(contracts)
}
