package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5String(inp string) string {
	h := md5.New()
	h.Write([]byte(inp))
	return hex.EncodeToString(h.Sum(nil))
}
