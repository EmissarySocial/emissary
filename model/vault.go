package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Created with help from:
// https://pkg.go.dev/crypto/cipher#NewGCM
// https://www.twilio.com/en-us/blog/encrypt-and-decrypt-data-in-go-with-aes-256

// Vault secures sensitive data in any model object
type Vault struct {
	Encrypted mapof.String `json:"-" bson:"encrypted"` // Encrypted vault data (generated from plaintet when saved)
	Nonce     string       `json:"-" bson:"nonce"`     // Nonce used to encrypt the vault data
	plaintext mapof.String `json:"-" bson:"-"`
}

// NewVault returns a fully initialized Vault object
func NewVault() Vault {

	return Vault{
		Encrypted: mapof.String{},
		plaintext: mapof.String{},
	}
}

func (vault Vault) GetStringOK(name string) (string, bool) {

	if _, ok := vault.Encrypted[name]; ok {
		return VaultObscuredValue, true
	}

	return "", false
}

func (vault *Vault) SetString(name string, value string) bool {

	if value == "" {
		delete(vault.plaintext, name)
		delete(vault.Encrypted, name)
		return true
	}

	if isEncryptable(value) {
		vault.plaintext[name] = value
	}

	return true
}

func (vault *Vault) Encrypt(encryptionKey []byte) error {

	const location = "model.vault.Encrypt"

	// If there are no plaintext values, then there is nothing to encrypt,
	// so lets save the work of setting up a block cipher and exit now.
	if !vault.hasEncryptableValues() {
		return nil
	}

	// Create AES block cipher
	block, err := aes.NewCipher(encryptionKey)

	if err != nil {
		return derp.Wrap(err, location, "Error creating AES block cipher")
	}

	// Create GCM
	aesgcm, err := cipher.NewGCM(block)

	if err != nil {
		return derp.Wrap(err, location, "Error generating GCM cipher")
	}

	// If not present, Create (and save) a randome nonce
	nonce := make([]byte, aesgcm.NonceSize())

	if len(vault.Nonce) == 0 {

		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return derp.Wrap(err, location, "Error generating nonce")
		}

		vault.Nonce = hex.EncodeToString(nonce)

	} else {
		nonce, err = hex.DecodeString(vault.Nonce)

		if err != nil {
			return derp.Wrap(err, location, "Invalid nonce in vault")
		}
	}

	// Encrypt all plaintext values in the vault
	for property, value := range vault.plaintext {

		if isEncryptable(value) {
			ciphertext := aesgcm.Seal(nil, nonce, []byte(value), nil)
			vault.Encrypted[property] = hex.EncodeToString(ciphertext)
		}
	}

	return nil
}

func (vault Vault) Decrypt(encryptionKey []byte) (mapof.String, error) {

	const location = "model.vault.Decrypt"

	if len(vault.Nonce) == 0 {
		return nil, derp.NewInternalError(location, "Nonce is not set")
	}

	// Create AES block cipher
	block, err := aes.NewCipher(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error creating AES block cipher")
	}

	// Create GCM
	aesgcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error generating GCM cipher")
	}

	// Retrieve the N-Once
	nonce, err := hex.DecodeString(vault.Nonce)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid nonce in vault")
	}

	// Decode ciphertext values
	result := make(mapof.String, len(vault.Encrypted))
	for property, value := range vault.Encrypted {

		ciphertext, err := hex.DecodeString(value)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid ciphertext in vault", property)
		}

		plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			panic(err.Error())
		}

		result[property] = string(plaintext)
	}

	// Success.
	return result, nil
}

// hasEncryptableValue returns TRUS if there are any non-empty/non-obscured values in the vault
// that should be encrypted
func (vault Vault) hasEncryptableValues() bool {

	for _, value := range vault.plaintext {
		if isEncryptable(value) {
			return true
		}
	}

	return false
}

// isEncryptable returns true if the value is not empty and not a `VaultObscuredValue`
func isEncryptable(value string) bool {
	if value == "" {
		return false
	}

	if value == VaultObscuredValue {
		return false
	}

	return true
}
