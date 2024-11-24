package qmc

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math"

	"golang.org/x/crypto/tea"
)

func simpleMakeKey(salt byte, length int) []byte {
	keyBuf := make([]byte, length)
	for i := 0; i < length; i++ {
		tmp := math.Tan(float64(salt) + float64(i)*0.1)
		keyBuf[i] = byte(math.Abs(tmp) * 100.0)
	}
	return keyBuf
}

const rawKeyPrefixV2 = "QQMusic EncV2,Key:"

func deriveKey(rawKey []byte) ([]byte, error) {
	rawKeyDec := make([]byte, base64.StdEncoding.DecodedLen(len(rawKey)))
	n, err := base64.StdEncoding.Decode(rawKeyDec, rawKey)
	if err != nil {
		return nil, err
	}
	rawKeyDec = rawKeyDec[:n]

	if bytes.HasPrefix(rawKeyDec, []byte(rawKeyPrefixV2)) {
		rawKeyDec, err = deriveKeyV2(bytes.TrimPrefix(rawKeyDec, []byte(rawKeyPrefixV2)))
		if err != nil {
			return nil, fmt.Errorf("deriveKeyV2 failed: %w", err)
		}
	}
	return deriveKeyV1(rawKeyDec)
}

func deriveKeyV1(rawKeyDec []byte) ([]byte, error) {
	if len(rawKeyDec) < 16 {
		return nil, errors.New("key length is too short")
	}

	simpleKey := simpleMakeKey(106, 8)
	teaKey := make([]byte, 16)
	for i := 0; i < 8; i++ {
		teaKey[i<<1] = simpleKey[i]
		teaKey[i<<1+1] = rawKeyDec[i]
	}

	rs, err := decryptTencentTea(rawKeyDec[8:], teaKey)
	if err != nil {
		return nil, err
	}
	return append(rawKeyDec[:8], rs...), nil
}

var (
	deriveV2Key1 = []byte{
		0x33, 0x38, 0x36, 0x5A, 0x4A, 0x59, 0x21, 0x40,
		0x23, 0x2A, 0x24, 0x25, 0x5E, 0x26, 0x29, 0x28,
	}

	deriveV2Key2 = []byte{
		0x2A, 0x2A, 0x23, 0x21, 0x28, 0x23, 0x24, 0x25,
		0x26, 0x5E, 0x61, 0x31, 0x63, 0x5A, 0x2C, 0x54,
	}
)

func deriveKeyV2(raw []byte) ([]byte, error) {
	buf, err := decryptTencentTea(raw, deriveV2Key1)
	if err != nil {
		return nil, err
	}

	buf, err = decryptTencentTea(buf, deriveV2Key2)
	if err != nil {
		return nil, err
	}

	n, err := base64.StdEncoding.Decode(buf, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func decryptTencentTea(inBuf []byte, key []byte) ([]byte, error) {
	const saltLen = 2
	const zeroLen = 7
	if len(inBuf)%8 != 0 {
		return nil, errors.New("inBuf size not a multiple of the block size")
	}
	if len(inBuf) < 16 {
		return nil, errors.New("inBuf size too small")
	}

	blk, err := tea.NewCipherWithRounds(key, 32)
	if err != nil {
		return nil, err
	}

	destBuf := make([]byte, 8)
	blk.Decrypt(destBuf, inBuf)
	padLen := int(destBuf[0] & 0x7)
	outLen := len(inBuf) - 1 - padLen - saltLen - zeroLen

	out := make([]byte, outLen)

	ivPrev := make([]byte, 8)
	ivCur := inBuf[:8]

	inBufPos := 8

	destIdx := 1 + padLen
	cryptBlock := func() {
		ivPrev = ivCur
		ivCur = inBuf[inBufPos : inBufPos+8]

		xor8Bytes(destBuf, destBuf, inBuf[inBufPos:inBufPos+8])
		blk.Decrypt(destBuf, destBuf)

		inBufPos += 8
		destIdx = 0
	}
	for i := 1; i <= saltLen; {
		if destIdx < 8 {
			destIdx++
			i++
		} else if destIdx == 8 {
			cryptBlock()
		}
	}

	outPos := 0
	for outPos < outLen {
		if destIdx < 8 {
			out[outPos] = destBuf[destIdx] ^ ivPrev[destIdx]
			destIdx++
			outPos++
		} else if destIdx == 8 {
			cryptBlock()
		}
	}

	for i := 1; i <= zeroLen; i++ {
		if destBuf[destIdx] != ivPrev[destIdx] {
			return nil, errors.New("zero check failed")
		}
	}

	return out, nil
}

func xor8Bytes(dst, a, b []byte) {
	for i := 0; i < 8; i++ {
		dst[i] = a[i] ^ b[i]
	}
}
