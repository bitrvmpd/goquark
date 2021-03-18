package usb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/bitrvmpd/goquark/internal/pkg/cfg"
	fsUtil "github.com/bitrvmpd/goquark/internal/pkg/fs"
)

type ID uint8

const (
	BlockSize = 4096
	GLCI      = 1229147207
	GLCO      = 1329810503
	header    = `
	####################################
	###### < < G O  Q U A R K > > ######
	####################################
	
	goQuark is ready for connections...
	
	+-----------------------------------+
	|	Client:		%v    |
	|	Version:	%v       |
	+-----------------------------------+

`
)

const (
	Invalid ID = iota
	GetDriveCount
	GetDriveInfo
	StatPath
	GetFileCount
	GetFile
	GetDirectoryCount
	GetDirectory
	StartFile
	ReadFile
	WriteFile
	EndFile
	Create
	Delete
	Rename
	GetSpecialPathCount
	GetSpecialPath
	SelectFile
)

type command struct {
	cmdMap map[ID]func()
	*buffer
}

func New(ctx context.Context) (*command, error) {
	c := command{
		buffer: &buffer{
			usbBuffer: &USBInterface{
				ctx: ctx,
			},
			inner_block: make([]byte, BlockSize),
			resp_block:  make([]byte, 0, BlockSize)},
	}

	// Map cmd ID to respective function
	c.cmdMap = map[ID]func(){
		Invalid:             func() { log.Printf("usbUtils.Invalid:") },
		GetDriveCount:       c.SendDriveCount,
		GetDriveInfo:        c.SendDriveInfo,
		StatPath:            func() { log.Printf("usbUtils.StatPath:") },
		GetFileCount:        func() { log.Printf("usbUtils.GetFileCount:") },
		GetFile:             func() { log.Printf("usbUtils.GetFile:") },
		GetDirectoryCount:   c.SendDirectoryCount,
		GetDirectory:        func() { log.Printf("usbUtils.GetDirectory:") },
		StartFile:           func() { log.Printf("usbUtils.StartFile:") },
		ReadFile:            func() { log.Printf("usbUtils.ReadFile:") },
		WriteFile:           func() { log.Printf("usbUtils.WriteFile:") },
		EndFile:             func() { log.Printf("usbUtils.EndFile:") },
		Create:              func() { log.Printf("usbUtils.Create:") },
		Delete:              func() { log.Printf("usbUtils.Delete:") },
		Rename:              func() { log.Printf("usbUtils.Rename:") },
		GetSpecialPathCount: c.SendSpecialPathCount,
		GetSpecialPath:      func() { log.Printf("usbUtils.SendSpecialPath:") },
		SelectFile:          c.SendSelectFile,
	}

	return &c, nil
}

func (c *command) ProcessUSBPackets() {

	// Check if device is connected.
	b := c.usbBuffer.isConnected()

	// Waits for device to appear
	// If false, returns.
	if !<-b {
		return
	}

	//quarkVersion := "0.4.0"
	//minGoldleafVersion := "0.8.0"

	// Reads goldleaf description
	d, err := c.retrieveDesc()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// Reads goldleaf's version number
	s, err := c.retrieveSerialNumber()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	fmt.Printf(header, d, s)

	for {
		// Magic [:4]
		i, err := c.readInt32()
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		if i != GLCI {
			log.Fatalf("ERROR: Invalid magic GLCI, got %v", i)
		}

		// CMD [4:]
		cmd, err := c.readCMD()
		if err != nil {
			log.Fatalln(err)
		}

		// Invoke requested function
		c.cmdMap[cmd]()
	}
}

func (c *command) readCMD() (ID, error) {
	if c.inner_block == nil {
		return Invalid, errors.New("ERROR: inner_block is not initialized")
	}

	return ID(binary.LittleEndian.Uint32(c.inner_block[4:])), nil
}

func (c *command) retrieveDesc() (string, error) {
	s, err := c.usbBuffer.getDescription()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) retrieveSerialNumber() (string, error) {
	s, err := c.usbBuffer.getSerialNumber()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) SendDriveCount() {
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	c.responseStart()
	c.writeInt32(uint32(len(drives)))
	c.responseEnd()
}

func (c *command) SendDriveInfo() {
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	_, err = c.readInt32() // It's in inner_block
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	idx := binary.LittleEndian.Uint32(c.inner_block[:4])

	if int(idx) > len(drives) || int(idx) <= -1 {
		c.respondFailure(0xDEAD)
		log.Fatalf("ERROR: Invalid disk index %v", idx)
	}

	drive := drives[idx]
	label, err := fsUtil.GetDriveLabel(drive)
	if err != nil {
		log.Fatalf("ERROR: Can't get drive label for %v", drive)
	}

	c.responseStart()
	c.writeString(label)
	c.writeString(drive)
	c.writeInt32(0) // It's in inner_block
	c.writeInt32(0) // It's in inner_block
	c.responseEnd()
}

func (c *command) SendSpecialPathCount() {
	c.responseStart()
	c.writeInt32(cfg.Size())
	c.responseEnd()
}

func (c *command) SendDirectoryCount() {
	s, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Can't send directory count for %v", err)
	}
	path := fsUtil.DenormalizePath(s)
	count, err := fsUtil.GetDirectoriesIn(path)
	if err != nil {
		log.Fatalf("ERROR: Can't get directories inside path %v", err)
	}
	c.responseStart()
	c.writeInt32(uint32(len(count)))
	c.responseEnd()
}

func (c *command) SendSelectFile() {
	path := fsUtil.NormalizePath("/Users/wuff/Documents/quarkgo")
	c.responseStart()
	c.writeString(path)
	c.responseEnd()
}
