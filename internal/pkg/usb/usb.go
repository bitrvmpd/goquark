package usbUtils

import (
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
}

func (u *USBInterface) GetDescription() (string, error) {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := ctx.OpenDeviceWithVIDPID(VendorID, ProductID)
	if err != nil {
		return "", err
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

	// Write data to the USB device.
	numBytes, err := ep.Write(p)
	if numBytes != len(p) {
		log.Fatalf("%s.Write([%v]): only %d bytes written, returned error is %v", ep, numBytes, numBytes, err)
	}

	return numBytes, nil
}
