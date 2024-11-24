package sniff

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/exp/slices"
)

type Sniffer interface {
	Sniff(header []byte) bool
}

var audioExtensions = map[string]Sniffer{
	// ref: https://mimesniff.spec.whatwg.org
	".mp3": prefixSniffer("ID3"), // todo: check mp3 without ID3v2 tag
	".ogg": prefixSniffer("OggS"),
	".wav": prefixSniffer("RIFF"),

	// ref: https://www.loc.gov/preservation/digital/formats/fdd/fdd000027.shtml
	".wma": prefixSniffer{
		0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11,
		0xa6, 0xd9, 0x00, 0xaa, 0x00, 0x62, 0xce, 0x6c,
	},

	// ref: https://www.garykessler.net/library/file_sigs.html
	".m4a": m4aSniffer{},    // MPEG-4 container, Apple Lossless Audio Codec
	".mp4": &mpeg4Sniffer{}, // MPEG-4 container, other fallback

	".flac": prefixSniffer("fLaC"), // ref: https://xiph.org/flac/format.html
	".dff":  prefixSniffer("FRM8"), // DSDIFF, ref: https://www.sonicstudio.com/pdf/dsd/DSDIFF_1.5_Spec.pdf

}

// AudioExtension sniffs the known audio types, and returns the file extension.
// header is recommended to at least 16 bytes.
func AudioExtension(header []byte) (string, bool) {
	for ext, sniffer := range audioExtensions {
		if sniffer.Sniff(header) {
			return ext, true
		}
	}
	return "", false
}

// AudioExtensionWithFallback is equivalent to AudioExtension, but returns fallback
// most likely to use .mp3 as fallback, because mp3 files may not have ID3v2 tag.
func AudioExtensionWithFallback(header []byte, fallback string) string {
	ext, ok := AudioExtension(header)
	if !ok {
		return fallback
	}
	return ext
}

type prefixSniffer []byte

func (s prefixSniffer) Sniff(header []byte) bool {
	return bytes.HasPrefix(header, s)
}

type m4aSniffer struct{}

func (m4aSniffer) Sniff(header []byte) bool {
	box := readMpeg4FtypBox(header)
	if box == nil {
		return false
	}

	return box.majorBrand == "M4A " || slices.Contains(box.compatibleBrands, "M4A ")
}

type mpeg4Sniffer struct{}

func (s *mpeg4Sniffer) Sniff(header []byte) bool {
	return readMpeg4FtypBox(header) != nil
}

type mpeg4FtpyBox struct {
	majorBrand       string
	minorVersion     uint32
	compatibleBrands []string
}

func readMpeg4FtypBox(header []byte) *mpeg4FtpyBox {
	if (len(header) < 8) || !bytes.Equal([]byte("ftyp"), header[4:8]) {
		return nil // not a valid ftyp box
	}

	size := binary.BigEndian.Uint32(header[0:4]) // size
	if size < 16 || size%4 != 0 {
		return nil // invalid ftyp box
	}

	box := mpeg4FtpyBox{
		majorBrand:   string(header[8:12]),
		minorVersion: binary.BigEndian.Uint32(header[12:16]),
	}

	// compatible brands
	for i := 16; i < int(size) && i+4 < len(header); i += 4 {
		box.compatibleBrands = append(box.compatibleBrands, string(header[i:i+4]))
	}

	return &box
}
