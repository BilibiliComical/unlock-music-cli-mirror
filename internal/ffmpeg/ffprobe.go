package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"

	"github.com/samber/lo"
)

type Result struct {
	Format  *Format   `json:"format"`
	Streams []*Stream `json:"streams"`
}

func (r *Result) HasAttachedPic() bool {
	return lo.ContainsBy(r.Streams, func(s *Stream) bool {
		return s.CodecType == "video"
	})
}

func (r *Result) getTagByKey(key string) string {
	for k, v := range r.Format.Tags {
		if key == strings.ToLower(k) {
			return v
		}
	}

	for _, stream := range r.Streams { // try to find in streams
		if stream.CodecType != "audio" {
			continue
		}
		for k, v := range stream.Tags {
			if key == strings.ToLower(k) {
				return v
			}
		}
	}
	return ""
}
func (r *Result) GetTitle() string {
	return r.getTagByKey("title")
}

func (r *Result) GetAlbum() string {
	return r.getTagByKey("album")
}

func (r *Result) GetArtists() []string {
	artists := strings.Split(r.getTagByKey("artist"), "/")
	for i := range artists {
		artists[i] = strings.TrimSpace(artists[i])
	}
	return artists
}

func (r *Result) HasMetadata() bool {
	return r.GetTitle() != "" || r.GetAlbum() != "" || len(r.GetArtists()) > 0
}

type Format struct {
	Filename       string            `json:"filename"`
	NbStreams      int               `json:"nb_streams"`
	NbPrograms     int               `json:"nb_programs"`
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	StartTime      string            `json:"start_time"`
	Duration       string            `json:"duration"`
	BitRate        string            `json:"bit_rate"`
	ProbeScore     int               `json:"probe_score"`
	Tags           map[string]string `json:"tags"`
}

type Stream struct {
	Index          int               `json:"index"`
	CodecName      string            `json:"codec_name"`
	CodecLongName  string            `json:"codec_long_name"`
	CodecType      string            `json:"codec_type"`
	CodecTagString string            `json:"codec_tag_string"`
	CodecTag       string            `json:"codec_tag"`
	SampleFmt      string            `json:"sample_fmt"`
	SampleRate     string            `json:"sample_rate"`
	Channels       int               `json:"channels"`
	ChannelLayout  string            `json:"channel_layout"`
	BitsPerSample  int               `json:"bits_per_sample"`
	RFrameRate     string            `json:"r_frame_rate"`
	AvgFrameRate   string            `json:"avg_frame_rate"`
	TimeBase       string            `json:"time_base"`
	StartPts       int               `json:"start_pts"`
	StartTime      string            `json:"start_time"`
	BitRate        string            `json:"bit_rate"`
	Disposition    *ProbeDisposition `json:"disposition"`
	Tags           map[string]string `json:"tags"`
}

type ProbeDisposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
	TimedThumbnails int `json:"timed_thumbnails"`
	Captions        int `json:"captions"`
	Descriptions    int `json:"descriptions"`
	Metadata        int `json:"metadata"`
	Dependent       int `json:"dependent"`
	StillImage      int `json:"still_image"`
}

func ProbeReader(ctx context.Context, rd io.Reader) (*Result, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet", // disable logging
		"-print_format", "json", // use json format
		"-show_format", "-show_streams", "-show_error", // retrieve format and streams
		"pipe:0", // input from stdin
	)

	cmd.Stdin = rd
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	ret := new(Result)
	if err := json.Unmarshal(stdout.Bytes(), ret); err != nil {
		return nil, err
	}

	return ret, nil
}
