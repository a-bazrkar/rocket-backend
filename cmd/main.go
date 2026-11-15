package main

import (
	"os"

	"github.com/a-bazrkar/rocket-backend/cmd/migrate"
	"github.com/a-bazrkar/rocket-backend/cmd/server"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:           "rocket",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// add commands
	root.AddCommand(
		server.Command(),
		migrate.Command(),
	)

	// execute root command
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
