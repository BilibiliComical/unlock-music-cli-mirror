package ncm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/utils"
)

const magicHeader = "CTENFDAM"

var (
	keyCore = []byte{
		0x68, 0x7a, 0x48, 0x52, 0x41, 0x6d, 0x73, 0x6f,
		0x35, 0x6b, 0x49, 0x6e, 0x62, 0x61, 0x78, 0x57,
	}
	keyMeta = []byte{
		0x23, 0x31, 0x34, 0x6C, 0x6A, 0x6B, 0x5F, 0x21,
		0x5C, 0x5D, 0x26, 0x30, 0x55, 0x3C, 0x27, 0x28,
	}
)

func NewDecoder(p *common.DecoderParams) common.Decoder {
	return &Decoder{rd: p.Reader, logger: p.Logger.With(zap.String("module", "ncm"))}
}

type Decoder struct {
	logger *zap.Logger
	rd     io.ReadSeeker // rd is the original file reader

	offset int
	cipher common.StreamDecoder

	metaRaw  []byte
	metaType string
	meta     ncmMeta
	cover    []byte
}

// Validate checks if the file is a valid Netease .ncm file.
// rd will be seeked to the beginning of the encrypted audio.
func (d *Decoder) Validate() error {
	if err := d.validateMagicHeader(); err != nil {
		return err
	}

	if _, err := d.rd.Seek(2, io.SeekCurrent); err != nil { // 2 bytes gap
		return fmt.Errorf("ncm seek file: %w", err)
	}

	keyData, err := d.readKeyData()
	if err != nil {
		return err
	}

	if err := d.readMetaData(); err != nil {
		return fmt.Errorf("read meta date failed: %w", err)
	}

	if _, err := d.rd.Seek(5, io.SeekCurrent); err != nil { // 5 bytes gap
		return fmt.Errorf("ncm seek gap: %w", err)
	}

	if err := d.readCoverData(); err != nil {
		return fmt.Errorf("parse ncm cover file failed: %w", err)
	}

	if err := d.parseMeta(); err != nil {
		return fmt.Errorf("parse meta failed: %w (raw json=%s)", err, string(d.metaRaw))
	}

	d.cipher = newNcmCipher(keyData)
	return nil
}

func (d *Decoder) validateMagicHeader() error {
	header := make([]byte, len(magicHeader)) // 0x00 - 0x07
	if _, err := d.rd.Read(header); err != nil {
		return fmt.Errorf("ncm read magic header: %w", err)
	}

	if !bytes.Equal([]byte(magicHeader), header) {
		return errors.New("ncm magic header not match")
	}

	return nil
}

func (d *Decoder) readKeyData() ([]byte, error) {
	bKeyLen := make([]byte, 4) //
	if _, err := io.ReadFull(d.rd, bKeyLen); err != nil {
		return nil, fmt.Errorf("ncm read key length: %w", err)
	}
	iKeyLen := binary.LittleEndian.Uint32(bKeyLen)

	bKeyRaw := make([]byte, iKeyLen)
	if _, err := io.ReadFull(d.rd, bKeyRaw); err != nil {
		return nil, fmt.Errorf("ncm read key data: %w", err)
	}
	for i := uint32(0); i < iKeyLen; i++ {
		bKeyRaw[i] ^= 0x64
	}

	return utils.PKCS7UnPadding(utils.DecryptAES128ECB(bKeyRaw, keyCore))[17:], nil
}

func (d *Decoder) readMetaData() error {
	bMetaLen := make([]byte, 4) //
	if _, err := io.ReadFull(d.rd, bMetaLen); err != nil {
		return fmt.Errorf("ncm read key length: %w", err)
	}
	iMetaLen := binary.LittleEndian.Uint32(bMetaLen)

	if iMetaLen == 0 {
		return nil // no meta data
	}

	bMetaRaw := make([]byte, iMetaLen)
	if _, err := io.ReadFull(d.rd, bMetaRaw); err != nil {
		return fmt.Errorf("ncm read meta data: %w", err)
	}
	bMetaRaw = bMetaRaw[22:] // skip "163 key(Don't modify):"
	for i := 0; i < len(bMetaRaw); i++ {
		bMetaRaw[i] ^= 0x63
	}

	cipherText, err := base64.StdEncoding.DecodeString(string(bMetaRaw))
	if err != nil {
		return errors.New("decode ncm meta failed: " + err.Error())
	}
	metaRaw := utils.PKCS7UnPadding(utils.DecryptAES128ECB(cipherText, keyMeta))
	sep := bytes.IndexByte(metaRaw, ':')
	if sep == -1 {
		return errors.New("invalid ncm meta file")
	}

	d.metaType = string(metaRaw[:sep])
	d.metaRaw = metaRaw[sep+1:]

	return nil
}

func (d *Decoder) readCoverData() error {
	bCoverFrameLen := make([]byte, 4)
	if _, err := io.ReadFull(d.rd, bCoverFrameLen); err != nil {
		return fmt.Errorf("ncm read cover length: %w", err)
	}

	coverFrameStartOffset, err := d.rd.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("ncm fetch cover frame start offset: %w", err)
	}
	coverFrameLen := binary.LittleEndian.Uint32(bCoverFrameLen)

	bCoverLen := make([]byte, 4)
	if _, err := io.ReadFull(d.rd, bCoverLen); err != nil {
		return fmt.Errorf("ncm read cover length: %w", err)
	}
	iCoverLen := binary.LittleEndian.Uint32(bCoverLen)

	coverBuf := make([]byte, iCoverLen)
	if _, err := io.ReadFull(d.rd, coverBuf); err != nil {
		return fmt.Errorf("ncm read cover data: %w", err)
	}
	d.cover = coverBuf

	offsetAudioData := coverFrameStartOffset + int64(coverFrameLen) + 4
	_, err = d.rd.Seek(offsetAudioData, io.SeekStart)

	return err
}

func (d *Decoder) parseMeta() error {
	switch d.metaType {
	case "music":
		d.meta = newNcmMetaMusic(d.logger)
		return json.Unmarshal(d.metaRaw, d.meta)
	case "dj":
		d.meta = new(ncmMetaDJ)
		return json.Unmarshal(d.metaRaw, d.meta)
	default:
		return errors.New("unknown ncm meta type: " + d.metaType)
	}
}

func (d *Decoder) Read(buf []byte) (int, error) {
	n, err := d.rd.Read(buf)
	if n > 0 {
		d.cipher.Decrypt(buf[:n], d.offset)
		d.offset += n
	}
	return n, err
}

func (d *Decoder) GetAudioExt() string {
	if d.meta != nil {
		if format := d.meta.GetFormat(); format != "" {
			return "." + d.meta.GetFormat()
		}
	}
	return ""
}

func (d *Decoder) GetCoverImage(ctx context.Context) ([]byte, error) {
	if d.cover != nil {
		return d.cover, nil
	}

	if d.meta == nil {
		return nil, errors.New("ncm meta not found")
	}
	imgURL := d.meta.GetAlbumImageURL()
	if !strings.HasPrefix(imgURL, "http") {
		return nil, nil // no cover image
	}

	// fetch cover image
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ncm download image failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ncm download image failed: unexpected http status %s", resp.Status)
	}
	d.cover, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ncm download image failed: %w", err)
	}

	return d.cover, nil
}

func (d *Decoder) GetAudioMeta(_ context.Context) (common.AudioMeta, error) {
	return d.meta, nil
}

func init() {
	// Netease Mp3/Flac
	common.RegisterDecoder("ncm", false, NewDecoder)
}
