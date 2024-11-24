package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type searchParams struct {
	Grp         int    `json:"grp"`
	NumPerPage  int    `json:"num_per_page"`
	PageNum     int    `json:"page_num"`
	Query       string `json:"query"`
	RemotePlace string `json:"remoteplace"`
	SearchType  int    `json:"search_type"`
	//SearchID    string `json:"searchid"` // todo: it seems generated randomly
}

type searchResponse struct {
	Body struct {
		Song struct {
			List []*TrackInfo `json:"list"`
		} `json:"song"`
	} `json:"body"`
	Code int `json:"code"`
}

func (c *QQMusic) Search(ctx context.Context, keyword string) ([]*TrackInfo, error) {

	resp, err := c.rpcCall(ctx,
		"music.search.SearchCgiService",
		"DoSearchForQQMusicDesktop",
		"music.search.SearchCgiService",
		&searchParams{
			SearchType: 0, Query: keyword,
			PageNum: 1, NumPerPage: 40,

			// static values
			Grp: 1, RemotePlace: "sizer.newclient.song",
		})
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[Search] rpc call: %w", err)
	}

	respData := searchResponse{}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("qqMusicClient[Search] unmarshal response: %w", err)
	}

	return respData.Body.Song.List, nil

}
