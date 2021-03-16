package quark

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitrvmpd/goquark/internal/pkg/usb"
)

func Listen() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c, err := usb.New()

	if err != nil {
		log.Fatalf("ERROR: Couldn't initialize command interface: %v", err)
	}

	go c.ProcessUSBPackets(ctx)

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range channel {
			fmt.Printf("\nClosing...\n")
			cancel()
		}
	}()

	// Wait for exit
	<-ctx.Done()
}
