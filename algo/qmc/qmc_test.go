package qmc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"unlock-music.dev/cli/algo/common"
)

func loadTestDataQmcDecoder(filename string) ([]byte, []byte, error) {
	encBody, err := os.ReadFile(fmt.Sprintf("./testdata/%s_raw.bin", filename))
	if err != nil {
		return nil, nil, err
	}
	encSuffix, err := os.ReadFile(fmt.Sprintf("./testdata/%s_suffix.bin", filename))
	if err != nil {
		return nil, nil, err
	}

	target, err := os.ReadFile(fmt.Sprintf("./testdata/%s_target.bin", filename))
	if err != nil {
		return nil, nil, err
	}
	return bytes.Join([][]byte{encBody, encSuffix}, nil), target, nil

}
func TestMflac0Decoder_Read(t *testing.T) {
	tests := []struct {
		name    string
		fileExt string
		wantErr bool
	}{
		{"mflac0_rc4", ".mflac0", false},
		{"mflac_rc4", ".mflac", false},
		{"mflac_map", ".mflac", false},
		{"mgg_map", ".mgg", false},
		{"qmc0_static", ".qmc0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, target, err := loadTestDataQmcDecoder(tt.name)
			if err != nil {
				t.Fatal(err)
			}

			d := NewDecoder(&common.DecoderParams{
				Reader:    bytes.NewReader(raw),
				Extension: tt.fileExt,
			})
			if err := d.Validate(); err != nil {
				t.Errorf("validate file error = %v", err)
			}

			buf := make([]byte, len(target))
			if _, err := io.ReadFull(d, buf); err != nil {
				t.Errorf("read bytes from decoder error = %v", err)
				return
			}
			if !reflect.DeepEqual(buf, target) {
				t.Errorf("Decrypt() got = %v, want %v", buf[:32], target[:32])
			}
		})
	}

}

func TestMflac0Decoder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fileExt string
		wantErr bool
	}{
		{"mflac0_rc4", ".flac", false},
		{"mflac_map", ".flac", false},
		{"mgg_map", ".ogg", false},
		{"qmc0_static", ".mp3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, _, err := loadTestDataQmcDecoder(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			d := NewDecoder(&common.DecoderParams{
				Reader:    bytes.NewReader(raw),
				Extension: tt.fileExt,
			})

			if err := d.Validate(); err != nil {
				t.Errorf("read bytes from decoder error = %v", err)
				return
			}
		})
	}
}
