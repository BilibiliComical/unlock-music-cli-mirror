package client

import (
	"context"
	"fmt"
	"strconv"
)

func (c *QQMusic) AlbumCoverByID(ctx context.Context, albumID int) ([]byte, error) {
	u := fmt.Sprintf("https://imgcache.qq.com/music/photo/album/%s/albumpic_%s_0.jpg",
		strconv.Itoa(albumID%100),
		strconv.Itoa(albumID),
	)
	return c.downloadFile(ctx, u)
}

func (c *QQMusic) AlbumCoverByMediaID(ctx context.Context, mediaID string) ([]byte, error) {
	// original: https://y.gtimg.cn/music/photo_new/T002M000%s.jpg
	u := fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R500x500M000%s.jpg", mediaID)
	return c.downloadFile(ctx, u)
}
