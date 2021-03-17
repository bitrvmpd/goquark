package cmd

import (
	"context"

	"github.com/bitrvmpd/goquark/internal/pkg/quark"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts Goldleaf client in command line",
	Long: `Starts listening for Goldleaf connection and serves the specified folders.
	If no folders are specified it serves the current one`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		//ctx, cancel := context.WithCancel(ctx)
		quark.Listen(ctx)
	},
}
