package kgm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	vprHeader = []byte{
		0x05, 0x28, 0xBC, 0x96, 0xE9, 0xE4, 0x5A, 0x43,
		0x91, 0xAA, 0xBD, 0xD0, 0x7A, 0xF5, 0x36, 0x31,
	}
	kgmHeader = []byte{
		0x7C, 0xD5, 0x32, 0xEB, 0x86, 0x02, 0x7F, 0x4B,
		0xA8, 0xAF, 0xA6, 0x8E, 0x0F, 0xFF, 0x99, 0x14,
	}

	ErrKgmMagicHeader = errors.New("kgm magic header not matched")
)

// header is the header of a KGM file.
type header struct {
	MagicHeader    []byte // 0x00-0x0f: magic header
	AudioOffset    uint32 // 0x10-0x13: offset of audio data
	CryptoVersion  uint32 // 0x14-0x17: crypto version
	CryptoSlot     uint32 // 0x18-0x1b: crypto key slot
	CryptoTestData []byte // 0x1c-0x2b: crypto test data
	CryptoKey      []byte // 0x2c-0x3b: crypto key
}

func (h *header) FromFile(rd io.ReadSeeker) error {
	if _, err := rd.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("kgm seek start: %w", err)
	}

	buf := make([]byte, 0x3c)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return fmt.Errorf("kgm read header: %w", err)
	}

	return h.FromBytes(buf)
}

func (h *header) FromBytes(buf []byte) error {
	if len(buf) < 0x3c {
		return errors.New("invalid kgm header length")
	}

	h.MagicHeader = buf[:0x10]
	if !bytes.Equal(kgmHeader, h.MagicHeader) && !bytes.Equal(vprHeader, h.MagicHeader) {
		return ErrKgmMagicHeader
	}

	h.AudioOffset = binary.LittleEndian.Uint32(buf[0x10:0x14])
	h.CryptoVersion = binary.LittleEndian.Uint32(buf[0x14:0x18])
	h.CryptoSlot = binary.LittleEndian.Uint32(buf[0x18:0x1c])
	h.CryptoTestData = buf[0x1c:0x2c]
	h.CryptoKey = buf[0x2c:0x3c]

	return nil
}
