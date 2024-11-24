package ximalaya

import (
	_ "embed"
	"encoding/binary"
)

const x2mHeaderSize = 1024

var x2mKey = [...]byte{'x', 'm', 'l', 'y'}
var x2mScrambleTable = [x2mHeaderSize]uint16{}

//go:embed x2m_scramble_table.bin
var x2mScrambleTableBytes []byte

func init() {
	if len(x2mScrambleTableBytes) != 2*x2mHeaderSize {
		panic("invalid x2m scramble table")
	}
	for i := range x2mScrambleTable {
		x2mScrambleTable[i] = binary.LittleEndian.Uint16(x2mScrambleTableBytes[i*2:])
	}
}

// decryptX2MHeader decrypts the header of ximalaya .x2m file.
// make sure input src is 1024(x2mHeaderSize) bytes long.
func decryptX2MHeader(src []byte) []byte {
	dst := make([]byte, len(src))
	for dstIdx := range src {
		srcIdx := x2mScrambleTable[dstIdx]
		dst[dstIdx] = src[srcIdx] ^ x2mKey[dstIdx%len(x2mKey)]
	}
	return dst
}
