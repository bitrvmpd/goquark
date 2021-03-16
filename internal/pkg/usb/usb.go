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
	ctx context.Context
}

// Waits for a device to be connected that matches VID: 0x057E PID: 0x3000
func (u *USBInterface) isConnected() chan bool {
	// Set ticker to search for the required device
	ticker := time.NewTicker(500 * time.Millisecond)
	c := make(chan bool)

	go func() {
		fmt.Println("Waiting for USB device to appear...")
		// Initialize a new Context.
		gctx := gousb.NewContext()
		defer gctx.Close()

		for range ticker.C {
			if u.ctx.Err() != nil {
				fmt.Println("Stopping func: isConnected...")
				c <- false
				return
			}

			// Open any device with a given VID/PID using a convenience function.
			// If none is found, it returns nil and nil error
			dev, _ := gctx.OpenDeviceWithVIDPID(VendorID, ProductID)
			if dev != nil {
				// Device found, retry
				dev.Close()
				c <- true
				return
			}

		}
	}()
	return c
}

func (u *USBInterface) GetDescription() (string, error) {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()
	// Open any device with a given VID/PID using a convenience function.
	// If none is found, it returns nil and nil error
	dev, err := ctx.OpenDeviceWithVIDPID(VendorID, ProductID)
	if dev == nil && err == nil {
		return "", fmt.Errorf("Coulnd't find specified device VID: %v, PID: %v", VendorID, ProductID)
	}

	defer dev.Close()

	s, err := dev.Product()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (u *USBInterface) GetSerialNumber() (string, error) {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := ctx.OpenDeviceWithVIDPID(VendorID, ProductID)
	if err != nil {
		return "", err
	}
	defer dev.Close()

	s, err := dev.SerialNumber()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (u *USBInterface) Read(p []byte) (int, error) {

	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := ctx.OpenDeviceWithVIDPID(VendorID, ProductID)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}
	defer dev.Close()

	// Claim the default interface using a convenience function.
	// The default interface is always #0 alt #0 in the currently active
	// config.
	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", dev, err)
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
			if u.ctx.Err() != nil {
				done()
				dev.Close()
				ctx.Close()
				break
			}
			// TODO: Improve by using channels
			time.Sleep(1 * time.Millisecond)
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

	return numBytes, nil
}

func (u *USBInterface) Write(p []byte) (int, error) {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := ctx.OpenDeviceWithVIDPID(VendorID, ProductID)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}
	defer dev.Close()

	// Claim the default interface using a convenience function.
	// The default interface is always #0 alt #0 in the currently active
	// config.
	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", dev, err)
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
			if u.ctx.Err() != nil {
				done()
				dev.Close()
				ctx.Close()
				break
			}
			// TODO: Improve by using channels
			time.Sleep(1 * time.Millisecond)
		}
	}()
	// Write data to the USB device.
	numBytes, err := ep.Write(p)
	if numBytes != len(p) {
		log.Fatalf("%s.Write([%v]): only %d bytes written, returned error is %v", ep, numBytes, numBytes, err)
	}

	return numBytes, nil
}
