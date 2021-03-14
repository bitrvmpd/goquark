package usbUtils

import (
	"encoding/binary"
	"encoding/hex"
	"log"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type ID uint8

const (
	BlockSize = 0x1000
	GLCI      = "49434C47"
	GLCO      = "4F434C47"
)

const (
	Invalid ID = iota
	GetDriveCount
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
	return &command{
		inner_block: make([]byte, BlockSize),
		resp_block:  make([]byte, BlockSize),
		usbBuffer:   &USBInterface{},
	}
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

func (c *command) ReadString() (string, error) {
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

func (c *command) WriteString(v string) error {
	_, err := c.usbBuffer.Read(c.inner_block)
	if err != nil {
		return err
	}

	// Prepare encoder
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	t := transform.NewWriter(c.usbBuffer, enc)
	// Parse bytes to utf8
	_, err = t.Write([]byte(v))
	if err != nil {
		return err
	}

	return nil
}

func (c *command) ResponseStart() {
	d, err := hex.DecodeString(GLCO)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	_, err = c.usbBuffer.Write(d)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	_, err = c.usbBuffer.Write([]byte{0})
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func (c *command) ResponseEnd() {
	_, err := c.usbBuffer.Write(c.resp_block)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func (c *command) RespondFailure(r uint32) {
	d, err := hex.DecodeString(GLCO)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	_, err = c.usbBuffer.Write(d)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	b := []byte{}
	binary.LittleEndian.PutUint32(b, r)

	_, err = c.usbBuffer.Write(b)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	c.ResponseEnd()
}

func (c *command) RespondEmpty() {
	c.ResponseStart()
	c.ResponseEnd()
}
