package random

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

/*******************************************
 * Modified from original source code (MIT licensed) at:
 * https://blog.questionable.services/article/generating-secure-random-numbers-crypto-rand/
 *******************************************/

// GenerateBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateString(s int) (string, error) {
	b, err := GenerateBytes(s)
	return Base64URLEncode(b), err
}

// Base64URLEncode base64 encodes the given bytes in a URL-safe way
func Base64URLEncode(b []byte) string {
	result := base64.URLEncoding.EncodeToString(b)
	result = strings.ReplaceAll(result, "+", "-")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "=", "")

	return result
}
