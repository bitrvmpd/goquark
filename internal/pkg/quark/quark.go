package quark

import (
	"context"
	"log"

	"github.com/bitrvmpd/goquark/internal/pkg/usb"
)

func Listen(ctx context.Context, cancel context.CancelFunc) {
	c, err := usb.New(ctx)
	if err != nil {
		log.Fatalf("ERROR: Couldn't initialize command interface: %v", err)
	}

	// Start listening for USB Packets
	go c.ProcessUSBPackets()

	// Wait for exit
	<-ctx.Done()
}
