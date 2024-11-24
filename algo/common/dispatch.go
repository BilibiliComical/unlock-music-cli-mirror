package common

import (
	"io"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

type DecoderParams struct {
	Reader    io.ReadSeeker // required
	Extension string        // required, source extension, eg. ".mp3"

	FilePath string // optional, source file path

	Logger *zap.Logger // required
}
type NewDecoderFunc func(p *DecoderParams) Decoder

type DecoderFactory struct {
	noop   bool
	Suffix string
	Create NewDecoderFunc
}

var DecoderRegistry []DecoderFactory

func RegisterDecoder(ext string, noop bool, dispatchFunc NewDecoderFunc) {
	DecoderRegistry = append(DecoderRegistry,
		DecoderFactory{noop: noop, Create: dispatchFunc, Suffix: "." + strings.TrimPrefix(ext, ".")})
}

func GetDecoder(filename string, skipNoop bool) []DecoderFactory {
	var result []DecoderFactory
	// Some extensions contains multiple dots, e.g. ".kgm.flac", hence iterate
	//   all decoders for each extension.
	name := strings.ToLower(filepath.Base(filename))
	for _, dec := range DecoderRegistry {
		if !strings.HasSuffix(name, dec.Suffix) {
			continue
		}
		if skipNoop && dec.noop {
			continue
		}
		result = append(result, dec)
	}
	return result
}
