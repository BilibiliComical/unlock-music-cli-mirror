package qmc

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestSimpleMakeKey(t *testing.T) {
	expect := []byte{0x69, 0x56, 0x46, 0x38, 0x2b, 0x20, 0x15, 0x0b}
	t.Run("106,8", func(t *testing.T) {
		if got := simpleMakeKey(106, 8); !reflect.DeepEqual(got, expect) {
			t.Errorf("simpleMakeKey() = %v, want %v", got, expect)
		}
	})
}
func loadDecryptKeyData(name string) ([]byte, []byte, error) {
	keyRaw, err := os.ReadFile(fmt.Sprintf("./testdata/%s_key_raw.bin", name))
	if err != nil {
		return nil, nil, err
	}
	keyDec, err := os.ReadFile(fmt.Sprintf("./testdata/%s_key.bin", name))
	if err != nil {
		return nil, nil, err
	}
	return keyRaw, keyDec, nil
}
func TestDecryptKey(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"mflac0_rc4(512)", "mflac0_rc4", false},
		{"mflac_map(256)", "mflac_map", false},
		{"mflac_rc4(256)", "mflac_rc4", false},
		{"mgg_map(256)", "mgg_map", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, want, err := loadDecryptKeyData(tt.filename)
			if err != nil {
				t.Fatalf("load test data failed: %s", err)
			}
			got, err := deriveKey(raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("deriveKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("deriveKey() got = %v..., want %v...",
					string(got[:32]), string(want[:32]))
			}
		})
	}
}
