package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"slices"

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

	if _, ok := vault.plaintext[name]; ok {
		return VaultObscuredValue, true
	}

	return "", false
}

func (vault *Vault) SetString(name string, value string) bool {

	if vault.plaintext == nil {
		vault.plaintext = mapof.NewString()
	}

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

	if vault.plaintext == nil {
		vault.plaintext = mapof.NewString()
	}

	if vault.Encrypted == nil {
		vault.Encrypted = mapof.NewString()
	}

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

	// If not present, Create (and save) a randome n-once
	nonce, err := vault.getNonce(aesgcm)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving n-once for vault")
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

func (vault Vault) Decrypt(encryptionKey []byte, values ...string) (mapof.String, error) {

	const location = "model.vault.Decrypt"

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
	nonce, err := vault.getNonce(aesgcm)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid nonce in vault")
	}

	// Decode ciphertext values
	result := make(mapof.String, len(values))
	for property, value := range vault.Encrypted {

		// If values are specified, then only decrypt those values.
		if len(values) > 0 {
			if !slices.Contains(values, property) {
				continue
			}
		}

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

	// Patch plaintext values into the result.
	// If a property is in the plaintext, then it hasn't been encrypted/saved yet.
	// They're still valid to use, so put them in here, if applicable.
	for property, value := range vault.plaintext {

		// If values are specified, then only decrypt those values.
		if len(values) > 0 {
			if !slices.Contains(values, property) {
				continue
			}
		}

		result[property] = value
	}

	// Success.
	return result, nil
}

// hasEncryptableValue returns TRUE if there are any non-empty/non-obscured values in the vault
// that should be encrypted
func (vault Vault) hasEncryptableValues() bool {

	for _, value := range vault.plaintext {
		if isEncryptable(value) {
			return true
		}
	}

	return false
}

// getNonce retrieves (or generates) the N-Once used to encrypt the vault data.
func (vault *Vault) getNonce(aesgcm cipher.AEAD) ([]byte, error) {
	const location = "model.vault.getNonce"

	if len(vault.Nonce) == 0 {

		// Generate a random N-Once
		nonce := make([]byte, aesgcm.NonceSize())

		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, derp.Wrap(err, location, "Error generating nonce")
		}

		vault.Nonce = hex.EncodeToString(nonce)

		return nonce, nil
	}

	// Decode the n-once as a hex string
	nonce, err := hex.DecodeString(vault.Nonce)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid nonce in vault")
	}

	return nonce, nil
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
