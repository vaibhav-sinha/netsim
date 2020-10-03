package utils

import "encoding/hex"

func HexStringToBytes(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}
