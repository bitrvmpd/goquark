package usb

import (
	"bytes"
	"encoding/binary"
	"log"

	"golang.org/x/text/encoding/unicode"
)

type buffer struct {
	in_buff  bytes.Buffer
	out_buff bytes.Buffer

	usb *USBInterface
}

func (c *buffer) responseStart() {
	// Empty our out buffer
	c.out_buff.Reset()

	//Fast convertion to uint32
	d := make([]byte, 4)
	binary.LittleEndian.PutUint32(d, GLCO)

	//Append to our magic and 0 delimiter
	c.out_buff.Write(d)

	//Fast convertion to uint32
	d = make([]byte, 4)
	binary.LittleEndian.PutUint32(d, 0)

	//Append to our magic and 0 delimiter
	c.out_buff.Write(d)
}

func (c *buffer) responseEnd() {
	// Fill with 0 up to 4096 bytes
	d := make([]byte, BlockSize-c.out_buff.Len())
	c.out_buff.Write(d)

	// Write the buffer
	_, err := c.usb.Write(c.out_buff.Bytes())
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func (c *buffer) respondFailure(r uint32) {
	// Empty our out buffer
	c.out_buff.Reset()

	//Fast convertion to uint32
	d := make([]byte, 4)
	binary.LittleEndian.PutUint32(d, GLCO)
	c.out_buff.Write(d)

	// Append error
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, r)
	c.out_buff.Write(b)

	c.responseEnd()
}

func (c *buffer) respondEmpty() {
	c.responseStart()
	c.responseEnd()
}

func (c *buffer) readInt32() (int, error) {
	d := make([]byte, 4)
	_, err := c.in_buff.Read(d)
	if err != nil {
		log.Fatalf("ERROR: Couldn't read from buffer!. %v", err)
	}
	i := binary.LittleEndian.Uint32(d)
	return int(i), nil
}

func (c *buffer) readInt64() (int64, error) {
	d := make([]byte, 8)
	_, err := c.in_buff.Read(d)
	if err != nil {
		log.Fatalf("ERROR: Couldn't read from buffer!. %v", err)
	}
	i := binary.LittleEndian.Uint64(d)
	return int64(i), nil
}

func (c *buffer) readFromUSB() {
	c.in_buff.Reset()
	b := make([]byte, BlockSize)
	_, err := c.usb.Read(b)
	if err != nil {
		log.Fatalf("ERROR: Couldn't read from usb!. %v", err)
	}
	c.in_buff.Write(b)
}

func (c *buffer) writeInt32(n uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	c.out_buff.Write(b)
}
func (c *buffer) writeInt64(n uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	c.out_buff.Write(b)
}

func (c *buffer) readString() (string, error) {
	//Pop the size
	size, err := c.readInt32()
	if err != nil {
		return "", err
	}

	o := make([]byte, size)
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	_, _, err = enc.Transform(o, c.in_buff.Next(size*2), false)
	if err != nil {
		return "", err
	}

	// Convert num of bytes reported by enc.Transform
	s := string(o)

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
	c.out_buff.Write(o[:nDst])
}
