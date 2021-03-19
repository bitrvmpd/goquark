package usb

import (
	"context"
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
			usb: &USBInterface{
				ctx: ctx,
			},
		}}

	// Map cmd ID to respective function
	c.cmdMap = map[ID]func(){
		Invalid:             func() { log.Printf("usbUtils.Invalid:") },
		GetDriveCount:       c.GetDriveCount,
		GetDriveInfo:        c.GetDriveInfo,
		StatPath:            c.StatPath,
		GetFileCount:        c.GetFileCount,
		GetFile:             c.GetFile,
		GetDirectoryCount:   c.GetDirectoryCount,
		GetDirectory:        c.GetDirectory,
		StartFile:           func() { log.Printf("usbUtils.StartFile:") },
		ReadFile:            c.ReadFile,
		WriteFile:           func() { log.Printf("usbUtils.WriteFile:") },
		EndFile:             func() { log.Printf("usbUtils.EndFile:") },
		Create:              func() { log.Printf("usbUtils.Create:") },
		Delete:              c.Delete,
		Rename:              c.Rename,
		GetSpecialPathCount: c.GetSpecialPathCount,
		GetSpecialPath:      c.GetSpecialPath,
		SelectFile:          c.SelectFile,
	}

	return &c, nil
}

func (c *command) ProcessUSBPackets() {

	// Check if device is connected.
	b := c.usb.isConnected()

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
		c.readFromUSB()

		// Magic [:4]
		i, err := c.readInt32()
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		if i != GLCI {
			log.Fatalf("ERROR: Invalid magic GLCI, got %v", i)
		}

		// CMD [4:]
		cmd, err := c.readInt32()
		if err != nil {
			log.Fatalln(err)
		}

		// Invoke requested function
		c.cmdMap[ID(cmd)]()
	}
}

func (c *command) retrieveDesc() (string, error) {
	s, err := c.usb.getDescription()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) retrieveSerialNumber() (string, error) {
	s, err := c.usb.getSerialNumber()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) GetDriveCount() {
	log.Println("GetDriveCount")
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	c.responseStart()
	c.writeInt32(uint32(len(drives)))
	c.responseEnd()
}

func (c *command) GetDriveInfo() {
	log.Println("GetDriveInfo")
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// Read payload
	idx, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't retrieve next int32 %v", err)
	}

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

func (c *command) GetSpecialPath() {
	log.Println("GetSpecialPath")

	// Read payload
	idx, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't retrieve next int32 %v", err)
	}

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

func (c *command) GetSpecialPathCount() {
	log.Println("GetSpecialPathCount")
	c.responseStart()
	c.writeInt32(cfg.Size())
	c.responseEnd()
}

func (c *command) GetDirectoryCount() {
	log.Println("GetDirectoryCount")
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

func (c *command) SelectFile() {
	log.Println("SelectFile")
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
		log.Printf("ERROR: Couldn't get %v stats. %v", path, err)
		c.respondFailure(0xDEAD)
		return
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

func (c *command) GetFileCount() {
	log.Println("GetFileCount")
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

func (c *command) GetFile() {
	log.Println("GetFile")
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Can't read string from buffer. %v", err)
		return
	}
	// idx comes after the path
	idx, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't retrieve next int32 %v", err)
	}

	path = fsUtil.DenormalizePath(path)
	files, err := fsUtil.GetFilesIn(path)
	if err != nil {
		log.Fatalf("ERROR: Can't get files in %v. %v", path, err)
		return
	}

	if idx >= len(files) || idx < 0 {
		c.respondFailure(0xDEAD)
		return
	}

	c.responseStart()
	c.writeString(files[idx])
	c.responseEnd()
}

func (c *command) GetDirectory() {
	log.Println("GetDirectory")
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	path = fsUtil.DenormalizePath(path)

	idx, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	dirs, err := fsUtil.GetDirectoriesIn(path)
	if err != nil {
		log.Fatalf("ERROR: Couldn't get directories in %v. %v", path, err)
	}

	if idx > len(dirs) || idx < 0 {
		c.respondFailure(0xDEAD)
	}

	c.responseStart()
	c.writeString(dirs[idx])
	c.responseEnd()
}

func (c *command) ReadFile() {
	log.Println("ReadFile")
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	path = fsUtil.DenormalizePath(path)

	offset, err := c.readInt64()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	size, err := c.readInt64()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("ERROR: Couldn't open %v. %v", path, err)
	}

	_, err = file.Seek(offset, 1)
	if err != nil {
		log.Fatalf("ERROR: Couldn't seek %v to offset %v. %v", path, offset, err)
	}

	fbuffer := make([]byte, size)
	bRead, err := file.Read(fbuffer)
	if err != nil {
		log.Fatalf("ERROR: Couldn't read %v. %v", path, err)
	}

	c.responseStart()
	c.writeInt64(uint64(bRead))
	c.responseEnd()

	if _, err = c.usb.Write(fbuffer); err != nil {
		log.Fatalf("ERROR: Couldn't write %v.", err)
	}
}

func (c *command) Rename() {
	fType, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	path = fsUtil.DenormalizePath(path)

	newPath, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	newPath = fsUtil.DenormalizePath(newPath)

	if fType != 1 && fType != 2 {
		c.respondFailure(0xDEAD)
	}

	err = os.Rename(path, newPath)
	if err != nil {
		log.Fatalf("ERROR: Couldn't rename %v to %v. %v", path, newPath, err)
	}

	c.respondEmpty()
}

func (c *command) Delete() {
	fType, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	path = fsUtil.DenormalizePath(path)

	if fType != 1 && fType != 2 {
		c.respondFailure(0xDEAD)
	}

	err = os.RemoveAll(path)
	if err != nil {
		log.Fatalf("ERROR: Couldn't removeAll %v. %v", path, err)
	}

	c.respondEmpty()
}
