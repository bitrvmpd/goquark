package ui

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/bitrvmpd/goquark/internal/pkg/cfg"
	"github.com/bitrvmpd/goquark/internal/pkg/quark"
	"github.com/getlantern/systray"
	"github.com/sqweek/dialog"
)

var folders map[int]*systray.MenuItem
var ctx context.Context
var cancel context.CancelFunc

func Build() {
	ctx = context.Background()
	ctx, cancel = context.WithCancel(ctx)
	systray.Run(onReady, onExit)
}

func onReady() {
	folders = make(map[int]*systray.MenuItem, cfg.Size())
	started := false
	//systray.SetIcon(icon.Data)
	systray.SetTitle("goQuark")
	systray.SetTooltip("")
	mStart := systray.AddMenuItem("Start", "Starts communication")
	mStatus := systray.AddMenuItem("Client Stopped", "Show client status")
	mStatus.Disable()
	systray.AddSeparator()

	// Sets the icon of a menu item. Only available on Mac and Windows.
	systray.AddSeparator()
	mPath := systray.AddMenuItem("Add Folder...", "Exposes a new folder to Goldleaf")

	//Reads folders, add tickers to disable them or deleting
	mPaths := systray.AddMenuItem("Remove Folder", "Click to remove an exposed folder")

	// Reads configuration, sets routes
	for i, folder := range cfg.ListFolders() {
		folders[i] = mPaths.AddSubMenuItem(folder.Alias, folder.Path)
	}

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	//Listen for submenu items
	for c, f := range folders {
		go func(fIndex int, mi *systray.MenuItem) {
			for {
				select {
				case <-ctx.Done():
					fmt.Printf("Closing submenu threads: %v\n", fIndex)
					return
				case <-mi.ClickedCh:
					cfg.RemoveFolder(fIndex)
					// systray doesn't have a way to remove an item. Hiding is ok for now.
					mi.Hide()
				}
			}
		}(c, f)
	}
	// Set button actions
	for {
		select {
		case <-mQuit.ClickedCh:
			systray.Quit()

		case <-mStart.ClickedCh:
			if started {
				// Stops the client
				cancel()
				started = false
				mStart.SetTitle("Start")
				mStatus.SetTitle("Client Stopped")
				mPaths.Enable()
				continue
			}

			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			go quark.Listen(ctx)
			started = true
			mStart.SetTitle("Stop")
			mStatus.SetTitle("Ready for connection")
			mPaths.Disable()

		case <-mPath.ClickedCh:
			f, err := dialog.Directory().Browse()
			if err != nil {
				log.Printf("INFO: %v", err)
			}

			// Don't add a folder if the user didn't selected it.
			if f == "" {
				continue
			}
			mPaths.AddSubMenuItem(path.Base(f), f)
			cfg.AddFolder(path.Base(f), f)
		}
	}

}

func onExit() {
	if cancel == nil {
		log.Println("ERROR: Couldn't call cancel")
		return
	}
	cancel()
}
