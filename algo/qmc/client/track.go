package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/samber/lo"
)

type getTrackInfoParams struct {
	Ctx   int   `json:"ctx"`
	Ids   []int `json:"ids"`
	Types []int `json:"types"`
}

type getTrackInfoResponse struct {
	Tracks []*TrackInfo `json:"tracks"`
}

func (c *QQMusic) GetTracksInfo(ctx context.Context, songIDs []int) ([]*TrackInfo, error) {
	resp, err := c.rpcCall(ctx,
		"Protocol_UpdateSongInfo",
		"CgiGetTrackInfo",
		"music.trackInfo.UniformRuleCtrl",
		&getTrackInfoParams{Ctx: 0, Ids: songIDs, Types: []int{0}},
	)
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[GetTrackInfo] rpc call: %w", err)
	}
	respData := getTrackInfoResponse{}
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, fmt.Errorf("qqMusicClient[GetTrackInfo] unmarshal response: %w", err)
	}

	return respData.Tracks, nil
}

func (c *QQMusic) GetTrackInfo(ctx context.Context, songID int) (*TrackInfo, error) {
	tracks, err := c.GetTracksInfo(ctx, []int{songID})
	if err != nil {
		return nil, fmt.Errorf("qqMusicClient[GetTrackInfo] get tracks info: %w", err)
	}

	if len(tracks) == 0 {
		return nil, fmt.Errorf("qqMusicClient[GetTrackInfo] track not found")
	}

	return tracks[0], nil
}

type TrackSinger struct {
	Id    int    `json:"id"`
	Mid   string `json:"mid"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Type  int    `json:"type"`
	Uin   int    `json:"uin"`
	Pmid  string `json:"pmid"`
}
type TrackAlbum struct {
	Id       int    `json:"id"`
	Mid      string `json:"mid"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Pmid     string `json:"pmid"`
}
type TrackInfo struct {
	Id       int           `json:"id"`
	Type     int           `json:"type"`
	Mid      string        `json:"mid"`
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Subtitle string        `json:"subtitle"`
	Singer   []TrackSinger `json:"singer"`
	Album    TrackAlbum    `json:"album"`
	Mv       struct {
		Id    int    `json:"id"`
		Vid   string `json:"vid"`
		Name  string `json:"name"`
		Title string `json:"title"`
		Vt    int    `json:"vt"`
	} `json:"mv"`
	Interval   int    `json:"interval"`
	Isonly     int    `json:"isonly"`
	Language   int    `json:"language"`
	Genre      int    `json:"genre"`
	IndexCd    int    `json:"index_cd"`
	IndexAlbum int    `json:"index_album"`
	TimePublic string `json:"time_public"`
	Status     int    `json:"status"`
	Fnote      int    `json:"fnote"`
	File       struct {
		MediaMid      string        `json:"media_mid"`
		Size24Aac     int           `json:"size_24aac"`
		Size48Aac     int           `json:"size_48aac"`
		Size96Aac     int           `json:"size_96aac"`
		Size192Ogg    int           `json:"size_192ogg"`
		Size192Aac    int           `json:"size_192aac"`
		Size128Mp3    int           `json:"size_128mp3"`
		Size320Mp3    int           `json:"size_320mp3"`
		SizeApe       int           `json:"size_ape"`
		SizeFlac      int           `json:"size_flac"`
		SizeDts       int           `json:"size_dts"`
		SizeTry       int           `json:"size_try"`
		TryBegin      int           `json:"try_begin"`
		TryEnd        int           `json:"try_end"`
		Url           string        `json:"url"`
		SizeHires     int           `json:"size_hires"`
		HiresSample   int           `json:"hires_sample"`
		HiresBitdepth int           `json:"hires_bitdepth"`
		B30S          int           `json:"b_30s"`
		E30S          int           `json:"e_30s"`
		Size96Ogg     int           `json:"size_96ogg"`
		Size360Ra     []interface{} `json:"size_360ra"`
		SizeDolby     int           `json:"size_dolby"`
		SizeNew       []interface{} `json:"size_new"`
	} `json:"file"`
	Pay struct {
		PayMonth   int `json:"pay_month"`
		PriceTrack int `json:"price_track"`
		PriceAlbum int `json:"price_album"`
		PayPlay    int `json:"pay_play"`
		PayDown    int `json:"pay_down"`
		PayStatus  int `json:"pay_status"`
		TimeFree   int `json:"time_free"`
	} `json:"pay"`
	Action struct {
		Switch   int `json:"switch"`
		Msgid    int `json:"msgid"`
		Alert    int `json:"alert"`
		Icons    int `json:"icons"`
		Msgshare int `json:"msgshare"`
		Msgfav   int `json:"msgfav"`
		Msgdown  int `json:"msgdown"`
		Msgpay   int `json:"msgpay"`
		Switch2  int `json:"switch2"`
		Icon2    int `json:"icon2"`
	} `json:"action"`
	Ksong struct {
		Id  int    `json:"id"`
		Mid string `json:"mid"`
	} `json:"ksong"`
	Volume struct {
		Gain float64 `json:"gain"`
		Peak float64 `json:"peak"`
		Lra  float64 `json:"lra"`
	} `json:"volume"`
	Label       string   `json:"label"`
	Url         string   `json:"url"`
	Ppurl       string   `json:"ppurl"`
	Bpm         int      `json:"bpm"`
	Version     int      `json:"version"`
	Trace       string   `json:"trace"`
	DataType    int      `json:"data_type"`
	ModifyStamp int      `json:"modify_stamp"`
	Aid         int      `json:"aid"`
	Tid         int      `json:"tid"`
	Ov          int      `json:"ov"`
	Sa          int      `json:"sa"`
	Es          string   `json:"es"`
	Vs          []string `json:"vs"`
}

func (t *TrackInfo) GetArtists() []string {
	return lo.Map(t.Singer, func(v TrackSinger, i int) string {
		return v.Name
	})
}

func (t *TrackInfo) GetTitle() string {
	return t.Title
}

func (t *TrackInfo) GetAlbum() string {
	return t.Album.Name
}
