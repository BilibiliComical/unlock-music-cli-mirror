package qmc

import (
	bytes "bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

type MusicExTagV1 struct {
	SongID        uint32 // Song ID
	Unknown1      uint32 // unused & unknown
	Unknown2      uint32 // unused & unknown
	MediaID       string // Media ID
	MediaFileName string // real file name
	Unknown3      uint32 // unused; uninitialized memory?

	// 16 byte at the end of tag.
	// TagSize should be respected when parsing.
	TagSize    uint32 // 19.57: fixed value: 0xC0
	TagVersion uint32 // 19.57: fixed value: 0x01
	TagMagic   []byte // fixed value "musicex\0" (8 bytes)
}

func NewMusicExTag(f io.ReadSeeker) (*MusicExTagV1, error) {
	_, err := f.Seek(-16, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("musicex seek error: %w", err)
	}

	buffer := make([]byte, 16)
	bytesRead, err := f.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("get musicex error: %w", err)
	}
	if bytesRead != 16 {
		return nil, fmt.Errorf("MusicExV1: read %d bytes (expected %d)", bytesRead, 16)
	}

	tag := &MusicExTagV1{
		TagSize:    binary.LittleEndian.Uint32(buffer[0x00:0x04]),
		TagVersion: binary.LittleEndian.Uint32(buffer[0x04:0x08]),
		TagMagic:   buffer[0x04:0x0C],
	}

	if !bytes.Equal(tag.TagMagic, []byte("musicex\x00")) {
		return nil, errors.New("MusicEx magic mismatch")
	}
	if tag.TagVersion != 1 {
		return nil, errors.New(fmt.Sprintf("unsupported musicex tag version. expecting 1, got %d", tag.TagVersion))
	}

	if tag.TagSize < 0xC0 {
		return nil, errors.New(fmt.Sprintf("unsupported musicex tag size. expecting at least 0xC0, got 0x%02x", tag.TagSize))
	}

	buffer = make([]byte, tag.TagSize)
	bytesRead, err = f.Read(buffer)
	if err != nil {
		return nil, err
	}
	if uint32(bytesRead) != tag.TagSize {
		return nil, fmt.Errorf("MusicExV1: read %d bytes (expected %d)", bytesRead, tag.TagSize)
	}

	tag.SongID = binary.LittleEndian.Uint32(buffer[0x00:0x04])
	tag.Unknown1 = binary.LittleEndian.Uint32(buffer[0x04:0x08])
	tag.Unknown2 = binary.LittleEndian.Uint32(buffer[0x08:0x0C])
	tag.MediaID = readUnicodeTagName(buffer[0x0C:], 30*2)
	tag.MediaFileName = readUnicodeTagName(buffer[0x48:], 50*2)
	tag.Unknown3 = binary.LittleEndian.Uint32(buffer[0xAC:0xB0])
	return tag, nil
}

// readUnicodeTagName reads a buffer to maxLen.
// reconstruct text by skipping alternate char (ascii chars encoded in UTF-16-LE),
// until finding a zero or reaching maxLen.
func readUnicodeTagName(buffer []byte, maxLen int) string {
	builder := strings.Builder{}

	for i := 0; i < maxLen; i += 2 {
		chr := buffer[i]
		if chr != 0 {
			builder.WriteByte(chr)
		} else {
			break
		}
	}

	return builder.String()
}
