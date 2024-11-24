package qmc

import (
	"errors"
)

// A rc4Cipher is an instance of RC4 using a particular key.
type rc4Cipher struct {
	box  []byte
	key  []byte
	hash uint32
	n    int
}

// newRC4Cipher creates and returns a new rc4Cipher. The key argument should be the
// RC4 key, at least 1 byte and at most 256 bytes.
func newRC4Cipher(key []byte) (*rc4Cipher, error) {
	n := len(key)
	if n == 0 {
		return nil, errors.New("qmc/cipher_rc4: invalid key size")
	}

	var c = rc4Cipher{key: key, n: n}
	c.box = make([]byte, n)

	for i := 0; i < n; i++ {
		c.box[i] = byte(i)
	}

	var j = 0
	for i := 0; i < n; i++ {
		j = (j + int(c.box[i]) + int(key[i%n])) % n
		c.box[i], c.box[j] = c.box[j], c.box[i]
	}
	c.getHashBase()
	return &c, nil
}

func (c *rc4Cipher) getHashBase() {
	c.hash = 1
	for i := 0; i < c.n; i++ {
		v := uint32(c.key[i])
		if v == 0 {
			continue
		}
		nextHash := c.hash * v
		if nextHash == 0 || nextHash <= c.hash {
			break
		}
		c.hash = nextHash
	}
}

const (
	rc4SegmentSize      = 5120
	rc4FirstSegmentSize = 128
)

func (c *rc4Cipher) Decrypt(src []byte, offset int) {
	toProcess := len(src)
	processed := 0
	markProcess := func(p int) (finished bool) {
		offset += p
		toProcess -= p
		processed += p
		return toProcess == 0
	}

	if offset < rc4FirstSegmentSize {
		blockSize := toProcess
		if blockSize > rc4FirstSegmentSize-offset {
			blockSize = rc4FirstSegmentSize - offset
		}
		c.encFirstSegment(src[:blockSize], offset)
		if markProcess(blockSize) {
			return
		}
	}

	if offset%rc4SegmentSize != 0 {
		blockSize := toProcess
		if blockSize > rc4SegmentSize-offset%rc4SegmentSize {
			blockSize = rc4SegmentSize - offset%rc4SegmentSize
		}
		c.encASegment(src[processed:processed+blockSize], offset)
		if markProcess(blockSize) {
			return
		}
	}
	for toProcess > rc4SegmentSize {
		c.encASegment(src[processed:processed+rc4SegmentSize], offset)
		markProcess(rc4SegmentSize)
	}

	if toProcess > 0 {
		c.encASegment(src[processed:], offset)
	}
}
func (c *rc4Cipher) encFirstSegment(buf []byte, offset int) {
	for i := 0; i < len(buf); i++ {
		buf[i] ^= c.key[c.getSegmentSkip(offset+i)]
	}
}

func (c *rc4Cipher) encASegment(buf []byte, offset int) {
	box := make([]byte, c.n)
	copy(box, c.box)
	j, k := 0, 0

	skipLen := (offset % rc4SegmentSize) + c.getSegmentSkip(offset/rc4SegmentSize)
	for i := -skipLen; i < len(buf); i++ {
		j = (j + 1) % c.n
		k = (int(box[j]) + k) % c.n
		box[j], box[k] = box[k], box[j]
		if i >= 0 {
			buf[i] ^= box[(int(box[j])+int(box[k]))%c.n]
		}
	}
}
func (c *rc4Cipher) getSegmentSkip(id int) int {
	seed := int(c.key[id%c.n])
	idx := int64(float64(c.hash) / float64((id+1)*seed) * 100.0)
	return int(idx % int64(c.n))
}
