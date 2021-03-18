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

	log.Println("======== RESPONSE END ========")
	log.Println(c.resp_block)
	log.Println("======== RESPONSE END ========")

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
	// Trim trailing 0
	d = trimTrailing(d)
	c.resp_block = append(c.resp_block, d...)

	// Append error
	b := make([]byte, 0, BlockSize)
	binary.LittleEndian.PutUint32(b, r)
	// Trim trailing 0
	b = trimTrailing(b)
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
	sLength := len(v)
	c.writeInt32(uint32(sLength))
	o := make([]byte, BlockSize-len(c.resp_block))

	// Prepare encoder
	enc := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewEncoder()
	_, _, err := enc.Transform(o, d, true)

	if err != nil {
		log.Fatalf("ERROR: Can't write string: %v", err)
	}

	o = trimTrailing(o)
	c.resp_block = append(c.resp_block, o...)
}

func trimTrailing(b []byte) []byte {
	var c int
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			c = i + 1
			break
		}
	}
	return b[:c]
}
