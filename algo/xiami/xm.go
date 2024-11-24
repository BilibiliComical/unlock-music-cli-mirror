package xiami

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"unlock-music.dev/cli/algo/common"
)

var (
	magicHeader  = []byte{'i', 'f', 'm', 't'}
	magicHeader2 = []byte{0xfe, 0xfe, 0xfe, 0xfe}
	typeMapping  = map[string]string{
		" WAV": "wav",
		"FLAC": "flac",
		" MP3": "mp3",
		" A4M": "m4a",
	}
	ErrMagicHeader = errors.New("xm magic header not matched")
)

type Decoder struct {
	rd     io.ReadSeeker // rd is the original file reader
	offset int

	cipher    common.StreamDecoder
	outputExt string
}

func (d *Decoder) GetAudioExt() string {
	if d.outputExt != "" {
		return "." + d.outputExt

	}
	return ""
}

func NewDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{rd: p.Reader}
}

// Validate checks if the file is a valid xiami .xm file.
// rd will set to the beginning of the encrypted audio data.
func (d *Decoder) Validate() error {
	header := make([]byte, 16) // xm header is fixed to 16 bytes

	if _, err := io.ReadFull(d.rd, header); err != nil {
		return fmt.Errorf("xm read header: %w", err)
	}

	// 0x00 - 0x03 and 0x08 - 0x0B: magic header
	if !bytes.Equal(magicHeader, header[:4]) || !bytes.Equal(magicHeader2, header[8:12]) {
		return ErrMagicHeader
	}

	// 0x04 - 0x07: Audio File Type
	var ok bool
	d.outputExt, ok = typeMapping[string(header[4:8])]
	if !ok {
		return fmt.Errorf("xm detect unknown audio type: %s", string(header[4:8]))
	}

	// 0x0C - 0x0E, Encrypt Start At, LittleEndian Unit24
	encStartAt := uint32(header[12]) | uint32(header[13])<<8 | uint32(header[14])<<16

	// 0x0F, XOR Mask
	d.cipher = newXmCipher(header[15], int(encStartAt))

	return nil
}

func (d *Decoder) Read(p []byte) (int, error) {
	n, err := d.rd.Read(p)
	if n > 0 {
		d.cipher.Decrypt(p[:n], d.offset)
		d.offset += n
	}
	return n, err
}

func init() {
	// Xiami Wav/M4a/Mp3/Flac
	common.RegisterDecoder("xm", false, NewDecoder)
	// Xiami Typed Format
	common.RegisterDecoder("wav", false, NewDecoder)
	common.RegisterDecoder("mp3", false, NewDecoder)
	common.RegisterDecoder("flac", false, NewDecoder)
	common.RegisterDecoder("m4a", false, NewDecoder)
}
