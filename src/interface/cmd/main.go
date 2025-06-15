package main

import (
	"github.com/alex-galey/dokku-mcp/src/infrastructure/cli"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cmdConfig := &cli.CommandConfig{
		Version:   Version,
		BuildTime: BuildTime,
	}

	rootCmd := cli.CreateRootCommand(cmdConfig)
	rootCmd.AddCommand(cli.CreateVersionCommand(cmdConfig))

	cli.Execute(rootCmd)
}
