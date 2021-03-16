package usb

import (
	"encoding/binary"
	"log"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type buffer struct {
	inner_block []byte
	resp_block  []byte
	usbBuffer   *USBInterface
}

func (c *buffer) responseStart() {
	// Empty our out buffer
	c.resp_block = make([]byte, 0, BlockSize)

	//Fast convertion to uint32
	d := make([]byte, BlockSize)
	binary.LittleEndian.PutUint32(d, GLCO)

	//Append to our magic and 0 delimiter
	c.resp_block = append(c.resp_block, d[:4]...)
	c.resp_block = append(c.resp_block, 0)
}

func (c *buffer) responseEnd() {
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

func (c *buffer) respondFailure(r uint32) {
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

	c.responseEnd()
}

func (c *buffer) respondEmpty() {
	c.responseStart()
	c.responseEnd()
}

func (c *buffer) readInt32() (int, error) {
	c.inner_block = make([]byte, BlockSize)
	_, err := c.usbBuffer.Read(c.inner_block)
	if err != nil {
		return 0, err
	}
	i := binary.LittleEndian.Uint32(c.inner_block[:4])
	return int(i), nil
}

func (c *buffer) writeInt32(n uint32) {
	b := make([]byte, BlockSize-len(c.resp_block))
	binary.LittleEndian.PutUint32(b, n)
	c.resp_block = append(c.resp_block[:5], b...)
}

func (c *buffer) readString() (string, error) {
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

func (c *buffer) writeString(v string) {
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
