package common

import (
	"reflect"
	"testing"
)

func TestParseFilenameMeta(t *testing.T) {

	tests := []struct {
		name     string
		wantMeta AudioMeta
	}{
		{
			name:     "test1",
			wantMeta: &filenameMeta{title: "test1"},
		},
		{
			name:     "周杰伦 - 晴天.flac",
			wantMeta: &filenameMeta{artists: []string{"周杰伦"}, title: "晴天"},
		},
		{
			name:     "Alan Walker _ Iselin Solheim - Sing Me to Sleep.flac",
			wantMeta: &filenameMeta{artists: []string{"Alan Walker", "Iselin Solheim"}, title: "Sing Me to Sleep"},
		},
		{
			name:     "Christopher,Madcon - Limousine.flac",
			wantMeta: &filenameMeta{artists: []string{"Christopher", "Madcon"}, title: "Limousine"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMeta := ParseFilenameMeta(tt.name); !reflect.DeepEqual(gotMeta, tt.wantMeta) {
				t.Errorf("ParseFilenameMeta() = %v, want %v", gotMeta, tt.wantMeta)
			}
		})
	}
}
