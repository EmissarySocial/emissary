package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/benpate/derp"
)

// encryptAesgcm encrypts a push payload with the "aesgcm" encoding that mobile clients decrypt.
// It returns the ciphertext, the salt, and the server's one-time public key.
func encryptAesgcm(p256dh string, auth string, payload []byte) (ciphertext []byte, salt []byte, serverPublic []byte, err error) {

	const location = "service.encryptAesgcm"

	clientPublic, err := decodeBase64URL(p256dh)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Decoding p256dh")
	}

	authSecret, err := decodeBase64URL(auth)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Decoding auth")
	}

	// A fresh key pair and salt for every message
	curve := ecdh.P256()
	serverKey, err := curve.GenerateKey(rand.Reader)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Generating server key")
	}

	clientKey, err := curve.NewPublicKey(clientPublic)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Parsing client key")
	}

	shared, err := serverKey.ECDH(clientKey)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Computing shared secret")
	}

	salt = make([]byte, 16)

	if _, err := rand.Read(salt); err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Generating salt")
	}

	serverPublic = serverKey.PublicKey().Bytes()

	contentKey, nonce, err := aesgcmKeys(shared, authSecret, salt, clientPublic, serverPublic)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Deriving keys")
	}

	gcm, err := newAesgcmCipher(contentKey)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Creating cipher")
	}

	// Two zero bytes say "no padding" -- clients read them before the payload
	plaintext := append([]byte{0, 0}, payload...)

	return gcm.Seal(nil, nonce, plaintext, nil), salt, serverPublic, nil
}

// aesgcmKeys derives the content key and nonce from the ECDH shared secret, as mobile clients do
func aesgcmKeys(shared []byte, authSecret []byte, salt []byte, clientPublic []byte, serverPublic []byte) (contentKey []byte, nonce []byte, err error) {

	const location = "service.aesgcmKeys"

	keyMaterial, err := hkdf.Key(sha256.New, shared, authSecret, "Content-Encoding: auth\x00", 32)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Deriving key material")
	}

	contentKey, err = hkdf.Key(sha256.New, keyMaterial, salt, aesgcmInfo("aesgcm", clientPublic, serverPublic), 16)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Deriving content key")
	}

	nonce, err = hkdf.Key(sha256.New, keyMaterial, salt, aesgcmInfo("nonce", clientPublic, serverPublic), 12)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Deriving nonce")
	}

	return contentKey, nonce, nil
}

// aesgcmInfo builds the HKDF info string: "Content-Encoding: <type>\0P-256\0" plus both keys with their lengths
func aesgcmInfo(contentType string, clientPublic []byte, serverPublic []byte) string {
	info := []byte("Content-Encoding: " + contentType + "\x00P-256\x00")
	info = append(info, 0, byte(len(clientPublic))) // #nosec G115 -- both keys are validated 65-byte P-256 points
	info = append(info, clientPublic...)
	info = append(info, 0, byte(len(serverPublic))) // #nosec G115 -- both keys are validated 65-byte P-256 points
	info = append(info, serverPublic...)
	return string(info)
}

// newAesgcmCipher returns an AES-GCM cipher for the provided content key
func newAesgcmCipher(contentKey []byte) (cipher.AEAD, error) {

	block, err := aes.NewCipher(contentKey)

	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

// decodeBase64URL decodes base64url with or without padding
func decodeBase64URL(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(value, "="))
}
