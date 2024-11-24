package qmc

import "errors"

type mapCipher struct {
	key  []byte
	box  []byte
	size int
}

func newMapCipher(key []byte) (*mapCipher, error) {
	if len(key) == 0 {
		return nil, errors.New("qmc/cipher_map: invalid key size")
	}
	c := &mapCipher{key: key, size: len(key)}
	c.box = make([]byte, c.size)
	return c, nil
}

func (c *mapCipher) getMask(offset int) byte {
	if offset > 0x7FFF {
		offset %= 0x7FFF
	}
	idx := (offset*offset + 71214) % c.size
	return c.rotate(c.key[idx], byte(idx)&0x7)
}

func (c *mapCipher) rotate(value byte, bits byte) byte {
	rotate := (bits + 4) % 8
	left := value << rotate
	right := value >> rotate
	return left | right
}

func (c *mapCipher) Decrypt(buf []byte, offset int) {
	for i := 0; i < len(buf); i++ {
		buf[i] ^= c.getMask(offset + i)
	}
}
