package kwm

import (
	"encoding/binary"
	"strconv"
)

type kwmCipher struct {
	mask []byte
}

func newKwmCipher(key []byte) *kwmCipher {
	return &kwmCipher{mask: generateMask(key)}
}

func generateMask(key []byte) []byte {
	keyInt := binary.LittleEndian.Uint64(key)
	keyStr := strconv.FormatUint(keyInt, 10)
	keyStrTrim := padOrTruncate(keyStr, 32)
	mask := make([]byte, 32)
	for i := 0; i < 32; i++ {
		mask[i] = keyPreDefined[i] ^ keyStrTrim[i]
	}
	return mask
}

func (c kwmCipher) Decrypt(buf []byte, offset int) {
	for i := range buf {
		buf[i] ^= c.mask[(offset+i)&0x1F] // equivalent: [i % 32]
	}
}
