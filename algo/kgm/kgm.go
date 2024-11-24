package kgm

import (
	"fmt"
	"io"

	"unlock-music.dev/cli/algo/common"
)

type Decoder struct {
	rd io.ReadSeeker

	cipher common.StreamDecoder
	offset int

	header header
}

func NewDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{rd: p.Reader}
}

// Validate checks if the file is a valid Kugou (.kgm, .vpr, .kgma) file.
// rd will be seeked to the beginning of the encrypted audio.
func (d *Decoder) Validate() (err error) {
	if err := d.header.FromFile(d.rd); err != nil {
		return err
	}
	// TODO; validate crypto version

	switch d.header.CryptoVersion {
	case 3:
		d.cipher, err = newKgmCryptoV3(&d.header)
		if err != nil {
			return fmt.Errorf("kgm init crypto v3: %w", err)
		}
	default:
		return fmt.Errorf("kgm: unsupported crypto version %d", d.header.CryptoVersion)
	}

	// prepare for read
	if _, err := d.rd.Seek(int64(d.header.AudioOffset), io.SeekStart); err != nil {
		return fmt.Errorf("kgm seek to audio: %w", err)
	}

	return nil
}

func (d *Decoder) Read(buf []byte) (int, error) {
	n, err := d.rd.Read(buf)
	if n > 0 {
		d.cipher.Decrypt(buf[:n], d.offset)
		d.offset += n
	}
	return n, err
}

func init() {
	// Kugou
	common.RegisterDecoder("kgm", false, NewDecoder)
	common.RegisterDecoder("kgma", false, NewDecoder)
	// Viper
	common.RegisterDecoder("vpr", false, NewDecoder)
	// Kugou Android
	common.RegisterDecoder("kgm.flac", false, NewDecoder)
	common.RegisterDecoder("vpr.flac", false, NewDecoder)
}
