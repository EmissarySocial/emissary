package hmac

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// Sign returns the HMAC signature of the message using the specified hash
func Sign(method string, secret string, message []byte) ([]byte, bool) {

	algorithm, ok := hashFactory(method)

	if !ok {
		return nil, false
	}

	mac := hmac.New(algorithm, []byte(secret))
	mac.Write(message)

	return mac.Sum(nil), true
}

// Validate returns TRUE if the signature matches the message+secret using the specified hash
func Validate(method string, secret string, message []byte, signature []byte) bool {

	algorithm, ok := hashFactory(method)

	if !ok {
		return false
	}

	mac := hmac.New(algorithm, []byte(secret))
	mac.Write(message)

	return hmac.Equal(signature, mac.Sum(nil))
}

// hashFactory returns a hash algorithm based on the method name.
// Recognized algorithms are: sha1, sha256, sha384, sha512.
// If the method is not supported, the second return value is false.
func hashFactory(method string) (func() hash.Hash, bool) {

	switch method {

	case "sha1":
		return sha1.New, true

	case "sha256":
		return sha256.New, true

	case "sha384":
		return sha512.New384, true

	case "sha512":
		return sha512.New, true

	default:
		return nil, false
	}
}
