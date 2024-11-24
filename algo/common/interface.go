package common

import (
	"context"
	"io"
)

type StreamDecoder interface {
	Decrypt(buf []byte, offset int)
}

type Decoder interface {
	Validate() error
	io.Reader
}

type CoverImageGetter interface {
	GetCoverImage(ctx context.Context) ([]byte, error)
}

type AudioMeta interface {
	GetArtists() []string
	GetTitle() string
	GetAlbum() string
}

type AudioMetaGetter interface {
	GetAudioMeta(ctx context.Context) (AudioMeta, error)
}
