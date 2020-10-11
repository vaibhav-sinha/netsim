package utils

import "encoding/hex"

func HexStringToBytes(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

func CalculateChecksum(data []byte) []byte {
	checksum := []byte{0}
	for _, d := range data {
		checksum[0] += d
	}

	return checksum
}
