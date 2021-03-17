package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitrvmpd/goquark/internal/pkg/quark"
	"github.com/getlantern/systray"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "quark",
	Short: "A golang implementation of Quark",
	Long: `
		Quark is Goldleaf's USB client`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		systray.Run(onReady, onExit)
	},
}

var ctx context.Context
var cancel context.CancelFunc

func onReady() {
	started := false
	//systray.SetIcon(icon.Data)
	systray.SetTitle("goQuark")
	systray.SetTooltip("")
	mStart := systray.AddMenuItem("Start", "Starts communication")

	//Reads folders, add tickers to disable them or deleting
	mPaths := systray.AddMenuItem("Folders", "Enable/Disable exposed folders")
	cName := mPaths.AddSubMenuItemCheckbox("custom Name", "full route", true)
	// Sets the icon of a menu item. Only available on Mac and Windows.
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	// Set button actions
	for {
		select {
		case <-mQuit.ClickedCh:
			if cancel != nil {
				cancel()
			}
			systray.Quit()
		case <-mStart.ClickedCh:
			if started {
				// Stops the client
				cancel()
				started = false
				mStart.SetTitle("Start")
				continue
			}
			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			go quark.Listen(ctx)
			started = true
			mStart.SetTitle("Stop")
		case <-cName.ClickedCh:
			fmt.Println("Some folder clicked")
		}
	}
}

func onExit() {
	// clean up here
}

func Execute() {
	// Exit when user press CTRL+C
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range channel {
			fmt.Printf("\nClosing...\n")
			cancel()
			systray.Quit()
			return
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
