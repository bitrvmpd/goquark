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

var folders map[string]*systray.MenuItem
var ctx context.Context
var cancel context.CancelFunc

func Build() {
	ctx = context.Background()
	ctx, cancel = context.WithCancel(ctx)
	systray.Run(onReady, onExit)
}

func onReady() {
	folders = make(map[string]*systray.MenuItem, cfg.Size())
	started := false
	//systray.SetIcon(icon.Data)
	systray.SetTitle("goQuark")
	systray.SetTooltip("")
	mStart := systray.AddMenuItem("Start", "Starts communication")

	//Reads folders, add tickers to disable them or deleting
	mPaths := systray.AddMenuItem("Folders", "Enable/Disable exposed folders")

	// Reads configuration, sets routes
	for folder, path := range cfg.ListFolders() {
		folders[folder] = mPaths.AddSubMenuItemCheckbox(folder, path, true)
	}
	// Sets the icon of a menu item. Only available on Mac and Windows.
	systray.AddSeparator()
	mPath := systray.AddMenuItem("Add Folder...", "Exposes a new folder to Goldleaf")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	//Listen for submenu items
	for c, f := range folders {
		go func(folder string, mi *systray.MenuItem) {
			for {
				select {
				case <-ctx.Done():
					fmt.Printf("Closing submenu threads: %v\n", folder)
					return
				case <-mi.ClickedCh:
					fmt.Printf("Clicked %v\n", folder)
					if mi.Checked() {
						mi.Uncheck()
					} else {
						mi.Check()
					}
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
				continue
			}

			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			go quark.Listen(ctx)
			started = true
			mStart.SetTitle("Stop")

		case <-mPath.ClickedCh:
			f, err := dialog.Directory().Browse()
			if err != nil {
				log.Printf("INFO: %v", err)
			}

			// Don't add a folder if the user didn't selected it.
			if f == "" {
				continue
			}

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
