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

var (
	fileReader *os.File
	fileWriter *os.File
)

const (
	BlockSize = 0x1000
	GLCI      = 0x49434C47
	GLCO      = 0x4F434C47
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
			usb: initDevice(ctx),
		}}

	// Map cmd ID to respective function
	c.cmdMap = map[ID]func(){
		Invalid:             func() { log.Printf("usbUtils.Invalid:") },
		GetDriveCount:       c.getDriveCount,
		GetDriveInfo:        c.getDriveInfo,
		StatPath:            c.statPath,
		GetFileCount:        c.getFileCount,
		GetFile:             c.getFile,
		GetDirectoryCount:   c.getDirectoryCount,
		GetDirectory:        c.getDirectory,
		StartFile:           c.startFile,
		ReadFile:            c.readFile,
		WriteFile:           c.writeFile,
		EndFile:             c.endFile,
		Create:              c.create,
		Delete:              c.delete,
		Rename:              c.rename,
		GetSpecialPathCount: c.getSpecialPathCount,
		GetSpecialPath:      c.getSpecialPath,
		SelectFile:          c.selectFile,
	}

	return &c, nil
}

func (c *command) ProcessUSBPackets() {

	// Loop waiting for device, improve by using recover someway
	for {
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

		// Loop for reading usb
		for {
			if err := c.readFromUSB(); err != nil {
				// When usb is disconnected don't panic.
				// I need to tell the program to wait for a device again.
				log.Printf("INFO: Lost connection to device. %v", err)
				log.Println("Exiting loop...")
				c.usb.Close()
				break
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
			cmd, err := c.readInt32()
			if err != nil {
				log.Fatalln(err)
			}

			// Invoke requested function
			c.cmdMap[ID(cmd)]()
		}
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

func (c *command) getDriveCount() {
	log.Println("GetDriveCount")
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	c.responseStart()
	c.writeInt32(uint32(len(drives)))
	c.responseEnd()
}

func (c *command) getDriveInfo() {
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

func (c *command) getSpecialPath() {
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

func (c *command) getSpecialPathCount() {
	log.Println("GetSpecialPathCount")
	c.responseStart()
	c.writeInt32(cfg.Size())
	c.responseEnd()
}

func (c *command) getDirectoryCount() {
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

func (c *command) selectFile() {
	log.Println("SelectFile")
	path := fsUtil.NormalizePath("/Users/wuff/Documents/quarkgo")
	c.responseStart()
	c.writeString(path)
	c.responseEnd()
}

func (c *command) statPath() {
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

func (c *command) getFileCount() {
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

func (c *command) getFile() {
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

func (c *command) getDirectory() {
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

// TODO: Follow "The happy path is left-aligned"
func (c *command) readFile() {
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

	var file *os.File

	if fileReader != nil {
		// Use the already opened fileReader
		file = fileReader
	} else {
		// Or Don't use it for some reason..
		file, err = os.Open(path)
		if err != nil {
			log.Fatalf("ERROR: Couldn't open %v. %v", path, err)
		}
	}

	_, err = file.Seek(offset, 0)
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

func (c *command) rename() {
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

func (c *command) delete() {
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

// TODO: Follow "The happy path is left-aligned"
func (c *command) create() {
	// 1 = file, 2 = dir
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

	// 1 = file, 2 = dir
	if fType == 1 {
		_, err := os.Create(path)
		if err != nil {
			log.Fatalf("ERROR: Couldn't create file %v. %v", path, err)
			c.respondFailure(0xDEAD)
			return
		}
	} else if fType == 2 {
		err := os.Mkdir(path, 0755)
		if err != nil {
			log.Fatalf("ERROR: Couldn't create file %v. %v", path, err)
			c.respondFailure(0xDEAD)
			return
		}
	}

	c.respondEmpty()
}

// TODO: Follow "The happy path is left-aligned"
func (c *command) endFile() {
	fMode, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	if fMode == 1 {
		if fileReader != nil {
			fileReader.Close()
			fileReader = nil
		}
	} else {
		if fileWriter != nil {
			fileWriter.Close()
			fileWriter = nil
		}
	}
	c.respondEmpty()
}

// TODO: Follow "The happy path is left-aligned"
func (c *command) startFile() {
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	path = fsUtil.DenormalizePath(path)

	fMode, err := c.readInt32()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	if fMode == 1 {
		if fileReader != nil {
			fileReader.Close()
		}
		// Open Read Only
		fileReader, err = os.Open(path)
		if err != nil {
			log.Fatalf("ERROR: Couldn't open %v. %v", path, err)
		}
	} else {
		if fileWriter != nil {
			fileWriter.Close()
		}
		//Open Read and Write
		fileWriter, err = os.Create(path)
		if err != nil {
			log.Fatalf("ERROR: Couldn't write %v. %v", path, err)
		}

		if fMode == 3 {
			fInfo, err := fileWriter.Stat()
			if err != nil {
				log.Fatalf("ERROR: Couldn't get stats for %v. %v", path, err)
			}
			_, err = fileWriter.Seek(fInfo.Size(), 0)
			if err != nil {
				log.Fatalf("ERROR: Couldn't get stats for %v. %v", path, err)
			}
		}
	}

	c.respondEmpty()
}

// TODO: Follow "The happy path is left-aligned"
func (c *command) writeFile() {
	path, err := c.readString()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read string from buffer. %v", err)
	}
	path = fsUtil.DenormalizePath(path)

	bLenght, err := c.readInt64()
	if err != nil {
		log.Fatalf("ERROR: Couldn't read int32 from buffer. %v", err)
	}

	buffer := make([]byte, bLenght)
	_, err = c.usb.Read(buffer)
	if err != nil {
		log.Fatalf("ERROR: Couldn't read directly from buffer. %v", err)
	}

	if fileWriter != nil {
		_, err := fileWriter.Write(buffer)
		if err != nil {
			log.Fatalf("ERROR: Couldn't write %v to disk. %v", path, err)
			c.respondFailure(0xDEAD)
			return
		}
		c.respondEmpty()
		return
	}
	err = os.WriteFile(path, buffer, os.ModeAppend)
	if err != nil {
		c.respondFailure(0xDEAD)
		return
	}
	c.respondEmpty()
}
