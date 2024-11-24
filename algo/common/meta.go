package common

import (
	"path"
	"strings"
)

type filenameMeta struct {
	artists []string
	title   string
	album   string
}

func (f *filenameMeta) GetArtists() []string {
	return f.artists
}

func (f *filenameMeta) GetTitle() string {
	return f.title
}

func (f *filenameMeta) GetAlbum() string {
	return f.album
}

func ParseFilenameMeta(filename string) (meta AudioMeta) {
	partName := strings.TrimSuffix(filename, path.Ext(filename))
	items := strings.Split(partName, "-")
	ret := &filenameMeta{}

	switch len(items) {
	case 0:
		// no-op
	case 1:
		ret.title = strings.TrimSpace(items[0])
	default:
		ret.title = strings.TrimSpace(items[len(items)-1])

		for _, v := range items[:len(items)-1] {
			artists := strings.FieldsFunc(v, func(r rune) bool {
				return r == ',' || r == '_'
			})
			for _, artist := range artists {
				ret.artists = append(ret.artists, strings.TrimSpace(artist))
			}
		}
	}

	return ret
}
