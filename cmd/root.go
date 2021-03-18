package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitrvmpd/goquark/internal/pkg/ui"
	"github.com/getlantern/systray"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goquark",
	Short: "A golang implementation of Quark",
	Long:  `GoQuark is Goldleaf's USB client`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Build()
	},
}

func Execute() {
	// Exit when user press CTRL+C
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range channel {
			systray.Quit()
			return
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
