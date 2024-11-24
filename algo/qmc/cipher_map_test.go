package qmc

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func loadTestDataMapCipher(name string) ([]byte, []byte, []byte, error) {
	key, err := os.ReadFile(fmt.Sprintf("./testdata/%s_key.bin", name))
	if err != nil {
		return nil, nil, nil, err
	}
	raw, err := os.ReadFile(fmt.Sprintf("./testdata/%s_raw.bin", name))
	if err != nil {
		return nil, nil, nil, err
	}
	target, err := os.ReadFile(fmt.Sprintf("./testdata/%s_target.bin", name))
	if err != nil {
		return nil, nil, nil, err
	}
	return key, raw, target, nil
}
func Test_mapCipher_Decrypt(t *testing.T) {

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"mflac_map", false},
		{"mgg_map", false},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			key, raw, target, err := loadTestDataMapCipher(tt.name)
			if err != nil {
				t.Fatalf("load testing data failed: %s", err)
			}
			c, err := newMapCipher(key)
			if err != nil {
				t.Errorf("init mapCipher failed: %s", err)
				return
			}
			c.Decrypt(raw, 0)
			if !reflect.DeepEqual(raw, target) {
				t.Error("overall")
			}
		})
	}
}
