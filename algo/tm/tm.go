package tm

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/sniff"
)

var replaceHeader = []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}
var magicHeader = []byte{0x51, 0x51, 0x4D, 0x55} //0x15, 0x1D, 0x1A, 0x21

type Decoder struct {
	raw io.ReadSeeker // raw is the original file reader

	offset int
	audio  io.Reader // audio is the decrypted audio data
}

func (d *Decoder) Validate() error {
	header := make([]byte, 8)
	if _, err := io.ReadFull(d.raw, header); err != nil {
		return fmt.Errorf("tm read header: %w", err)
	}

	if bytes.Equal(magicHeader, header[:len(magicHeader)]) { // replace m4a header
		d.audio = io.MultiReader(bytes.NewReader(replaceHeader), d.raw)
		return nil
	}

	if _, ok := sniff.AudioExtension(header); ok { // not encrypted
		d.audio = io.MultiReader(bytes.NewReader(header), d.raw)
		return nil
	}

	return errors.New("tm: valid magic header")
}

func (d *Decoder) Read(buf []byte) (int, error) {
	return d.audio.Read(buf)
}

func NewTmDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{raw: p.Reader}
}

func init() {
	// QQ Music IOS M4a (replace header)
	common.RegisterDecoder("tm2", false, NewTmDecoder)
	common.RegisterDecoder("tm6", false, NewTmDecoder)

	// QQ Music IOS Mp3 (not encrypted)
	common.RegisterDecoder("tm0", false, NewTmDecoder)
	common.RegisterDecoder("tm3", false, NewTmDecoder)
}
