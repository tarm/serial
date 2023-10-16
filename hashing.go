package serial

import (
	"crypto/sha256"
	"encoding/hex"
)

// 6 letter hash
func shortHash(str string) string {
	bs := sha256.Sum256([]byte(str))
	hash := hex.EncodeToString(bs[:])[:6]
	return hash
}
