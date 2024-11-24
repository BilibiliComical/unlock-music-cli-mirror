package ncm

type ncmCipher struct {
	key []byte
	box []byte
}

func newNcmCipher(key []byte) *ncmCipher {
	return &ncmCipher{
		key: key,
		box: buildKeyBox(key),
	}
}

func (c *ncmCipher) Decrypt(buf []byte, offset int) {
	for i := 0; i < len(buf); i++ {
		buf[i] ^= c.box[(i+offset)&0xff]
	}
}

func buildKeyBox(key []byte) []byte {
	box := make([]byte, 256)
	for i := 0; i < 256; i++ {
		box[i] = byte(i)
	}

	var j byte
	for i := 0; i < 256; i++ {
		j = box[i] + j + key[i%len(key)]
		box[i], box[j] = box[j], box[i]
	}

	ret := make([]byte, 256)
	var _i byte
	for i := 0; i < 256; i++ {
		_i = byte(i + 1)
		si := box[_i]
		sj := box[_i+si]
		ret[i] = box[si+sj]
	}
	return ret
}
