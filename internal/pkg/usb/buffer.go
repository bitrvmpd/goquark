package usb

import (
	"encoding/binary"
	"log"

	"golang.org/x/text/encoding/unicode"
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
	d := make([]byte, 4)
	binary.LittleEndian.PutUint32(d, GLCO)

	//Append to our magic and 0 delimiter
	c.resp_block = append(c.resp_block, d...)

	//Fast convertion to uint32
	d = make([]byte, 4)
	binary.LittleEndian.PutUint32(d, 0)

	//Append to our magic and 0 delimiter
	c.resp_block = append(c.resp_block, d...)
}

func (c *buffer) responseEnd() {
	// Fill with 0 up to 4096 bytes
	d := make([]byte, BlockSize-len(c.resp_block))
	c.resp_block = append(c.resp_block, d...)

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
	d := make([]byte, 0, 4)
	binary.LittleEndian.PutUint32(d, GLCO)
	c.resp_block = append(c.resp_block, d...)

	// Append error
	b := make([]byte, 0, 4)
	binary.LittleEndian.PutUint32(b, r)
	c.resp_block = append(c.resp_block, b...)

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
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	c.resp_block = append(c.resp_block, b...)
}
func (c *buffer) writeInt64(n uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	c.resp_block = append(c.resp_block, b...)
}

func (c *buffer) readString() (string, error) {
	//TODO: make it less verbose.
	str := make([]byte, BlockSize)
	// Get String length
	length := binary.LittleEndian.Uint32(c.inner_block[8:12])
	// Prepare decoder, skip magic + cmd + string length
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	nDst, _, err := enc.Transform(str, c.inner_block[12:12+(length*2)], false)
	if err != nil {
		return "", err
	}

	// Convert num of bytes reported by enc.Transform
	s := string(str[:nDst])
	return s, nil
}

func (c *buffer) writeString(v string) {
	o := make([]byte, BlockSize)

	// Prepare encoder
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	nDst, _, err := enc.Transform(o, []byte(v), false)

	//Write len of chars.
	c.writeInt32(uint32(len(v)))
	if err != nil {
		log.Fatalf("ERROR: Can't write string: %v", err)
	}
	// Write num of bytes reported by enc.Transform
	c.resp_block = append(c.resp_block, o[:nDst]...)
}
