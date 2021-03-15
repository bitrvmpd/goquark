package usbUtils

import (
	"encoding/binary"
	"errors"
	"log"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

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
	inner_block []byte
	resp_block  []byte
	usbBuffer   *USBInterface
}

func NewCommand() *command {
	c := command{
		usbBuffer:   &USBInterface{},
		inner_block: make([]byte, BlockSize),
		resp_block:  make([]byte, 0, BlockSize),
	}
	return &c
}

func (c *command) ReadCMD() (ID, error) {
	if c.inner_block == nil {
		return Invalid, errors.New("ERROR: inner_block is not initialized")
	}

	return ID(binary.LittleEndian.Uint32(c.inner_block[4:])), nil
}

func (c *command) RetrieveDesc() (string, error) {
	s, err := c.usbBuffer.GetDescription()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) RetrieveSerialNumber() (string, error) {
	s, err := c.usbBuffer.GetSerialNumber()
	if err != nil {
		return "", err
	}
	return s, nil
}

func (c *command) ResponseStart() {
	// Empty our out buffer
	c.resp_block = make([]byte, 0, BlockSize)

	//Fast convertion to uint32
	d := make([]byte, BlockSize)
	binary.LittleEndian.PutUint32(d, GLCO)

	//Append to our magic and 0 delimiter
	c.resp_block = append(c.resp_block, d[:4]...)
	c.resp_block = append(c.resp_block, 0)
}

func (c *command) ResponseEnd() {
	// Fill with 0 up to 4096 bytes
	d := make([]byte, BlockSize-len(c.resp_block))
	c.resp_block = append(c.resp_block, d...)

	log.Println("SENDING: ", len(c.resp_block), "bytes...")

	// Write the buffer
	_, err := c.usbBuffer.Write(c.resp_block)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func (c *command) RespondFailure(r uint32) {
	// Empty our out buffer
	c.resp_block = make([]byte, 0, BlockSize)

	// Append magic
	d := make([]byte, 0, BlockSize)
	binary.LittleEndian.PutUint32(d, GLCO)
	c.resp_block = append(c.resp_block, d[:4]...)

	// Append error
	b := make([]byte, 0, BlockSize)
	binary.LittleEndian.PutUint32(b, r)
	c.resp_block = append(c.resp_block, b[:4]...)

	c.ResponseEnd()
}

func (c *command) RespondEmpty() {
	c.ResponseStart()
	c.ResponseEnd()
}

func (c *command) ReadInt32() (int, error) {
	c.inner_block = make([]byte, BlockSize)
	_, err := c.usbBuffer.Read(c.inner_block)
	if err != nil {
		return 0, err
	}
	i := binary.LittleEndian.Uint32(c.inner_block[:4])
	return int(i), nil
}

func (c *command) WriteInt32(n uint32) {
	b := make([]byte, BlockSize-len(c.resp_block))
	binary.LittleEndian.PutUint32(b, n)
	c.resp_block = append(c.resp_block[:5], b...)
}

func (c *command) ReadString() (string, error) {
	c.inner_block = make([]byte, BlockSize)
	_, err := c.usbBuffer.Read(c.inner_block)
	if err != nil {
		return "", err
	}

	// Prepare decoder
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	t := transform.NewReader(c.usbBuffer, enc)
	// Parse bytes to utf8
	_, err = t.Read(c.inner_block)
	if err != nil {
		return "", err
	}

	s := string(c.inner_block)
	return s, nil
}

func (c *command) WriteString(v string) {
	d := []byte(v)
	o := make([]byte, BlockSize-len(c.resp_block))

	// Prepare encoder
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	_, _, err := enc.Transform(o, d, true)

	if err != nil {
		log.Fatalf("ERROR: Can't write string: %v", err)
	}

	c.resp_block = append(c.resp_block[:5], o...)
}

func (c *command) SendDriveCount() {
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	c.ResponseStart()
	c.WriteInt32(uint32(len(drives)))
	c.ResponseEnd()
}

func (c *command) SendDriveInfo() {
	drives, err := fsUtil.ListDrives()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	_, err = c.ReadInt32() // It's in inner_block
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	idx := binary.LittleEndian.Uint32(c.inner_block[:4])

	if int(idx) > len(drives) || int(idx) <= -1 {
		c.RespondFailure(0xDEAD)
		log.Fatalf("ERROR: Invalid disk index %v", idx)
	}

	drive := drives[idx]
	label, err := fsUtil.GetDriveLabel(drive)
	if err != nil {
		log.Fatalf("ERROR: Can't get drive label for %v", drive)
	}

	c.ResponseStart()
	c.WriteString(label)
	c.WriteString(drive)
	c.WriteInt32(0) // It's in inner_block
	c.WriteInt32(0) // It's in inner_block
	c.ResponseEnd()
}

func (c *command) SendSpecialPathCount() {
	c.ResponseStart()
	c.WriteInt32(cfg.Size())
	c.ResponseEnd()
}

func (c *command) SendDirectoryCount() {
	s, err := c.ReadString()
	if err != nil {
		log.Fatalf("ERROR: Can't send directory count for %v", err)
	}
	path := fsUtil.DenormalizePath(s)
	count, err := fsUtil.GetDirectoriesIn(path)
	if err != nil {
		log.Fatalf("ERROR: Can't get directories inside path %v", err)
	}
	c.ResponseStart()
	c.WriteInt32(uint32(len(count)))
	c.ResponseEnd()
}

func (c *command) SendSelectFile() {
	path := fsUtil.NormalizePath("/Users/wuff/Documents/quarkgo")
	c.ResponseStart()
	c.WriteString(path)
	c.ResponseEnd()
}
