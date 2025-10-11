package main

import (
	"fmt"
	"os"

	"github.com/dokku-mcp/dokku-mcp/pkg/fxapp"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Handle version flag before Fx starts
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("dokku-mcp version %s (built on %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	fxapp.New().Run()
}
