package tools

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func Hash(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}

	return true
}
