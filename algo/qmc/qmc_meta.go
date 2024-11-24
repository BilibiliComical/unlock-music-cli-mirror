package qmc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/algo/qmc/client"
	"unlock-music.dev/cli/internal/ffmpeg"
)

func (d *Decoder) GetAudioMeta(ctx context.Context) (common.AudioMeta, error) {
	if d.meta != nil {
		return d.meta, nil
	}

	if d.songID != 0 {
		if err := d.getMetaBySongID(ctx); err != nil {
			return nil, err
		}
		return d.meta, nil
	}

	embedMeta, err := ffmpeg.ProbeReader(ctx, d.probeBuf)
	if err != nil {
		return nil, fmt.Errorf("qmc[GetAudioMeta] probe reader: %w", err)
	}
	d.meta = embedMeta
	d.embeddedCover = embedMeta.HasAttachedPic()

	if !d.embeddedCover && embedMeta.HasMetadata() {
		if err := d.searchMetaOnline(ctx, embedMeta); err != nil {
			return nil, err
		}
		return d.meta, nil
	}

	return d.meta, nil
}

func (d *Decoder) getMetaBySongID(ctx context.Context) error {
	c := client.NewQQMusicClient() // todo: use global client
	trackInfo, err := c.GetTrackInfo(ctx, d.songID)
	if err != nil {
		return fmt.Errorf("qmc[GetAudioMeta] get track info: %w", err)
	}

	d.meta = trackInfo
	d.albumID = trackInfo.Album.Id
	if trackInfo.Album.Pmid == "" {
		d.albumMediaID = trackInfo.Album.Pmid
	} else {
		d.albumMediaID = trackInfo.Album.Mid
	}
	return nil
}

func (d *Decoder) searchMetaOnline(ctx context.Context, original common.AudioMeta) error {
	c := client.NewQQMusicClient() // todo: use global client
	keyword := lo.WithoutEmpty(append(
		[]string{original.GetTitle(), original.GetAlbum()},
		original.GetArtists()...),
	)
	if len(keyword) == 0 {
		return errors.New("qmc[searchMetaOnline] no keyword")
	}

	trackList, err := c.Search(ctx, strings.Join(keyword, " "))
	if err != nil {
		return fmt.Errorf("qmc[searchMetaOnline] search: %w", err)
	}

	if len(trackList) == 0 {
		return errors.New("qmc[searchMetaOnline] no result")
	}

	meta := trackList[0]
	d.meta = meta
	d.albumID = meta.Album.Id
	if meta.Album.Pmid == "" {
		d.albumMediaID = meta.Album.Pmid
	} else {
		d.albumMediaID = meta.Album.Mid
	}

	return nil
}

func (d *Decoder) GetCoverImage(ctx context.Context) ([]byte, error) {
	if d.cover != nil {
		return d.cover, nil
	}

	if d.embeddedCover {
		img, err := ffmpeg.ExtractAlbumArt(ctx, d.probeBuf)
		if err != nil {
			return nil, fmt.Errorf("qmc[GetCoverImage] extract album art: %w", err)
		}

		d.cover = img.Bytes()

		return d.cover, nil
	}

	c := client.NewQQMusicClient() // todo: use global client
	var err error

	if d.albumMediaID != "" {
		d.cover, err = c.AlbumCoverByMediaID(ctx, d.albumMediaID)
		if err != nil {
			return nil, fmt.Errorf("qmc[GetCoverImage] get cover by media id: %w", err)
		}
	} else if d.albumID != 0 {
		d.cover, err = c.AlbumCoverByID(ctx, d.albumID)
		if err != nil {
			return nil, fmt.Errorf("qmc[GetCoverImage] get cover by id: %w", err)
		}
	} else {
		return nil, errors.New("qmc[GetAudioMeta] album (or media) id is empty")
	}

	return d.cover, nil

}
