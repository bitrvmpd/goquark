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
	*buffer
}

func New() (*command, error) {
	c := command{
		buffer: &buffer{
			usbBuffer:   &USBInterface{},
			inner_block: make([]byte, BlockSize),
			resp_block:  make([]byte, 0, BlockSize)},
	}

	return &c, nil
}

func (c *command) ProcessUSBPackets(ctx context.Context) {

	// Check if device is connected.
	b := c.usbBuffer.IsConnected(ctx)

	// If false, returns
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

	fmt.Printf(
		`
###################################
######## < < Q U A R K > > ########
###################################

goQuark is ready for connections...

+-----------------------------------+
|	Client:		%v    |
|	Version:	%v       |
+-----------------------------------+
`, d, s)

	for {
		// Check if context was closed
		if ctx.Err() != nil {
			fmt.Println("Stop processing packets...")
			return
		}

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

		switch cmd {
		case Invalid:
			log.Printf("usbUtils.Invalid:")
		case GetDriveCount:
			log.Printf("usbUtils.SendDriveCount:")
			c.SendDriveCount()
		case GetDriveInfo:
			log.Printf("usbUtils.SendDriveInfo:")
			c.SendDriveInfo()
		case StatPath:
			log.Printf("usbUtils.StatPath:")
		case GetFileCount:
			log.Printf("usbUtils.GetFileCount:")
		case GetFile:
			log.Printf("usbUtils.GetFile:")
		case GetDirectoryCount:
			log.Printf("usbUtils.GetDirectoryCount:")
			c.SendDirectoryCount()
		case GetDirectory:
			log.Printf("usbUtils.GetDirectory:")
		case StartFile:
			log.Printf("usbUtils.StartFile:")
		case ReadFile:
			log.Printf("usbUtils.ReadFile:")
		case WriteFile:
			log.Printf("usbUtils.WriteFile:")
		case EndFile:
			log.Printf("usbUtils.EndFile:")
		case Create:
			log.Printf("usbUtils.Create:")
		case Delete:
			log.Printf("usbUtils.Delete:")
		case Rename:
			log.Printf("usbUtils.Rename:")
		case GetSpecialPathCount:
			log.Printf("usbUtils.SendSpecialPathCount:")
			c.SendSpecialPathCount()
		case GetSpecialPath:
			log.Printf("usbUtils.SendSpecialPath:")
		case SelectFile:
			log.Printf("usbUtils.SendSelectFile:")
			c.SendSelectFile()
		default:
			log.Printf("usbUtils.default:")
		}
	}
}

func (c *command) readCMD() (ID, error) {
	if c.inner_block == nil {
		return Invalid, errors.New("ERROR: inner_block is not initialized")
	}

	return ID(binary.LittleEndian.Uint32(c.inner_block[4:])), nil
}

func (c *command) retrieveDesc() (string, error) {
	s, err := c.usbBuffer.GetDescription()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) retrieveSerialNumber() (string, error) {
	s, err := c.usbBuffer.GetSerialNumber()
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
