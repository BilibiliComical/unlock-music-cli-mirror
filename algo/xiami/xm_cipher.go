package xiami

type xmCipher struct {
	mask           byte
	encryptStartAt int
}

func newXmCipher(mask byte, encryptStartAt int) *xmCipher {
	return &xmCipher{
		mask:           mask,
		encryptStartAt: encryptStartAt,
	}
}

func (c *xmCipher) Decrypt(buf []byte, offset int) {
	for i := 0; i < len(buf); i++ {
		if offset+i >= c.encryptStartAt {
			buf[i] ^= c.mask
		}
	}
}
