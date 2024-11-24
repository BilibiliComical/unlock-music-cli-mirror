package ximalaya

import (
	"bytes"
	"fmt"
	"io"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/sniff"
)

type Decoder struct {
	rd     io.ReadSeeker
	offset int

	audio io.Reader
}

func NewDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{rd: p.Reader}
}

func (d *Decoder) Validate() error {
	encryptedHeader := make([]byte, x2mHeaderSize)
	if _, err := io.ReadFull(d.rd, encryptedHeader); err != nil {
		return fmt.Errorf("ximalaya read header: %w", err)
	}

	{ // try to decode with x2m
		header := decryptX2MHeader(encryptedHeader)
		if _, ok := sniff.AudioExtension(header); ok {
			d.audio = io.MultiReader(bytes.NewReader(header), d.rd)
			return nil
		}
	}

	{ // try to decode with x3m
		// not read file again, since x2m and x3m have the same header size
		header := decryptX3MHeader(encryptedHeader)
		if _, ok := sniff.AudioExtension(header); ok {
			d.audio = io.MultiReader(bytes.NewReader(header), d.rd)
			return nil
		}
	}

	return fmt.Errorf("ximalaya: unknown format")
}

func (d *Decoder) Read(p []byte) (n int, err error) {
	return d.audio.Read(p)
}

func init() {
	common.RegisterDecoder("x2m", false, NewDecoder)
	common.RegisterDecoder("x3m", false, NewDecoder)
	common.RegisterDecoder("xm", false, NewDecoder)
}
