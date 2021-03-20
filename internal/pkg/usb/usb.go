package usb

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/gousb"
)

const (
	VendorID      = 0x057E
	ProductID     = 0x3000
	WriteEndpoint = 0x1
	ReadEndpoint  = 0x81
)

type USBInterface struct {
	ctx  context.Context
	gCtx *gousb.Context
	gDev *gousb.Device
}

func initDevice(ctx context.Context) *USBInterface {
	return &USBInterface{
		ctx: ctx,
	}
}

func (u *USBInterface) Close() {
	u.gDev.Close()
	u.gCtx.Close()
	log.Println("Closing gDev and gCtx")
}

// Waits for a device to be connected that matches VID: 0x057E PID: 0x3000
// Stores context and device for reuse.
func (u *USBInterface) isConnected() chan bool {
	// Set ticker to search for the required device
	ticker := time.NewTicker(500 * time.Millisecond)
	c := make(chan bool)

	go func() {
		fmt.Println("Waiting for USB device to appear...")

		// Initialize a new Context.
		gctx := gousb.NewContext()

		for range ticker.C {
			select {
			case <-u.ctx.Done():
				gctx.Close()
				c <- false
				return
			case <-ticker.C:
				// Open any device with a given VID/PID using a convenience function.
				// If none is found, it returns nil and nil error
				dev, _ := gctx.OpenDeviceWithVIDPID(VendorID, ProductID)
				if dev != nil {
					// Device found, exit loop. don't close it!
					u.gCtx = gctx
					u.gDev = dev
					c <- true
					return
				}
			}
		}
	}()
	return c
}

func (u *USBInterface) getDescription() (string, error) {
	s, err := u.gDev.Product()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (u *USBInterface) getSerialNumber() (string, error) {
	s, err := u.gDev.SerialNumber()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (u *USBInterface) Read(p []byte) (int, error) {
	//Test donde channel for each request.
	chDone := make(chan interface{})

	// Claim the default interface using a convenience function.
	// The default interface is always #0 alt #0 in the currently active
	// config.
	intf, done, err := u.gDev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", u.gDev, err)
	}
	defer done()

	// Open an IN endpoint.
	ep, err := intf.InEndpoint(ReadEndpoint)
	if err != nil {
		log.Fatalf("%s.OutEndpoint(%v): %v", intf, WriteEndpoint, err)
	}

	// Set transfer as bulk
	//ep.Desc.MaxPacketSize = BlockSize
	ep.Desc.TransferType = gousb.TransferTypeBulk
	ep.Desc.IsoSyncType = gousb.IsoSyncTypeSync
	ep.Desc.PollInterval = 0 * time.Millisecond

	// Just before reading, prepare a way to cancel this request
	go func() {
		for {
			select {
			case <-u.ctx.Done():
				done()
				u.Close()
				return
			case <-chDone:
				// Successful read, close this goroutine
				return
			}
		}
	}()
	// Read data from the USB device.
	numBytes, err := ep.Read(p)
	if err != nil {
		return 0, err
	}

	if numBytes != len(p) {
		log.Fatalf("%s.Read([%v]): only %d bytes read, returned error is %v", ep, numBytes, numBytes, err)
	}

	// Notify that we are done!
	chDone <- struct{}{}
	return numBytes, nil
}

func (u *USBInterface) Write(p []byte) (int, error) {
	//Test donde channel for each request.
	chDone := make(chan interface{})

	// Claim the default interface using a convenience function.
	// The default interface is always #0 alt #0 in the currently active
	// config.
	intf, done, err := u.gDev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", u.gDev, err)
	}
	defer done()

	// Open an OUT endpoint.
	ep, err := intf.OutEndpoint(WriteEndpoint)
	if err != nil {
		log.Fatalf("%s.OutEndpoint(%v): %v", intf, WriteEndpoint, err)
	}

	// Set transfer as bulk
	//ep.Desc.MaxPacketSize = BlockSize
	ep.Desc.TransferType = gousb.TransferTypeBulk
	ep.Desc.IsoSyncType = gousb.IsoSyncTypeSync
	ep.Desc.PollInterval = 0 * time.Millisecond

	// Just before writting, prepare a way to cancel this request
	go func() {
		for {
			select {
			case <-u.ctx.Done():
				done()
				u.Close()
				return
			case <-chDone:
				// Successful read, close this goroutine
				return
			}
		}
	}()
	// Write data to the USB device.
	numBytes, err := ep.Write(p)
	if numBytes != len(p) {
		log.Fatalf("%s.Write([%v]): only %d bytes written, returned error is %v", ep, numBytes, numBytes, err)
	}

	// Notify that we are done!
	chDone <- struct{}{}
	return numBytes, nil
}
