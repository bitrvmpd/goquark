package usb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"

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
		StatPath:            c.StatPath,
		GetFileCount:        c.SendFileCount,
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
		GetSpecialPath:      c.SendSpecialPath,
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
	log.Println("SendDriveCount")
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	c.responseStart()
	c.writeInt32(uint32(len(drives)))
	c.responseEnd()
}

func (c *command) SendDriveInfo() {
	log.Println("SendDriveInfo")
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// Read payload
	idx := binary.LittleEndian.Uint32(c.inner_block[8:])

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
	c.writeInt32(0)
	c.writeInt32(0)
	c.responseEnd()
}

func (c *command) SendSpecialPath() {
	log.Println("SendSpecialPath")

	// Read payload
	idx := binary.LittleEndian.Uint32(c.inner_block[8:12])

	if int(idx) > int(cfg.Size()) || int(idx) <= -1 {
		c.respondFailure(0xDEAD)
		log.Fatalf("ERROR: Invalid path index %v", idx)
	}

	folders := cfg.ListFolders()
	folder := folders[idx]

	c.responseStart()
	c.writeString(folder.Alias)
	c.writeString(fsUtil.NormalizePath(folder.Path))
	c.responseEnd()
}

func (c *command) SendSpecialPathCount() {
	log.Println("SendSpecialPathCount")
	c.responseStart()
	c.writeInt32(cfg.Size())
	c.responseEnd()
}

func (c *command) SendDirectoryCount() {
	log.Println("SendDirectoryCount")
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
	log.Println("SendSelectFile")
	path := fsUtil.NormalizePath("/Users/wuff/Documents/quarkgo")
	c.responseStart()
	c.writeString(path)
	c.responseEnd()
}

func (c *command) StatPath() {
	log.Println("StatPath")
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Can't read string from buffer. %v", err)
		return
	}

	path = fsUtil.DenormalizePath(path)
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatalf("ERROR: Couldn't get %v stats. %v", path, err)
	}

	ftype := 0
	var fsize int64 = 0

	if !fi.IsDir() {
		ftype = 1
		fsize = fi.Size()
	}
	if fi.IsDir() {
		ftype = 2
	}
	if ftype == 0 {
		c.respondFailure(0xDEAD)
		return
	}

	c.responseStart()
	c.writeInt32(uint32(ftype))
	c.writeInt64(uint64(fsize))
	c.responseEnd()
}

func (c *command) SendFileCount() {
	log.Println("SendFileCount")
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Can't read string from buffer. %v", err)
		return
	}
	path = fsUtil.DenormalizePath(path)
	nFiles, err := fsUtil.GetFilesIn(path)
	if err != nil {
		log.Fatalf("ERROR: Can't get files in %v. %v", path, err)
		return
	}

	c.responseStart()
	c.writeInt32(uint32(len(nFiles)))
	c.responseEnd()
}
