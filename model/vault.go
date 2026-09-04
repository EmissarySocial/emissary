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
	Encrypted mapof.String `json:"-" bson:"encrypted"` // Encrypted vault data (generated from plaintext when saved)
	Nonces    mapof.String `json:"-" bson:"nonces"`    // Nonce used to encrypt each value, keyed the same as `Encrypted`
	Nonce     string       `json:"-" bson:"nonce"`     // DEPRECATED. Shared nonce written by an earlier format; still read, never written.
	plaintext mapof.String `json:"-" bson:"-"`
}

// NewVault returns a fully initialized Vault object
func NewVault() Vault {

	return Vault{
		Encrypted: mapof.String{},
		Nonces:    mapof.String{},
		plaintext: mapof.String{},
	}
}

// HasString returns TRUE if the vault has a value for the specified name.
// This method is lightweight because it does not decrypt the value, only
// checks to see if it exists.
func (vault Vault) HasString(name string) bool {

	if _, ok := vault.Encrypted[name]; ok {
		return true
	}

	if _, ok := vault.plaintext[name]; ok {
		return true
	}

	return false
}

// GetStringOK returns human-visible value for the specified name,
// which means returning a `VaultObscuredValue` the value is present,
// or a blank string if it is not.
func (vault Vault) GetStringOK(name string) (string, bool) {

	if _, ok := vault.Encrypted[name]; ok {
		return VaultObscuredValue, true
	}

	if _, ok := vault.plaintext[name]; ok {
		return VaultObscuredValue, true
	}

	return "", false
}

// SetString sets the value for a specified name in the vault. It does
// this intelligently, by not overwriting actual values if a `VaultObscuredValue`
// was passed in through the UX.
func (vault *Vault) SetString(name string, value string) bool {

	if vault.plaintext == nil {
		vault.plaintext = mapof.NewString()
	}

	if value == "" {
		delete(vault.plaintext, name)
		delete(vault.Encrypted, name)

		// The nonce is meaningless without the ciphertext it opened, so it goes too.
		// Leaving it behind would accumulate orphans that outlive every value they described.
		delete(vault.Nonces, name)
		return true
	}

	if isEncryptable(value) {
		vault.plaintext[name] = value
	}

	return true
}

// Encrypt seals this Vault's plaintext values using the provided key
func (vault *Vault) Encrypt(encryptionKey []byte) error {

	const location = "model.vault.Encrypt"

	if vault.plaintext == nil {
		vault.plaintext = mapof.NewString()
	}

	if vault.Encrypted == nil {
		vault.Encrypted = mapof.NewString()
	}

	if vault.Nonces == nil {
		vault.Nonces = mapof.NewString()
	}

	// If there are no plaintext values, then there is nothing to encrypt,
	// so lets save the work of setting up a block cipher and exit now.
	if !vault.hasEncryptableValues() {
		return nil
	}

	// Create AES block cipher
	block, err := aes.NewCipher(encryptionKey)

	if err != nil {
		return derp.Wrap(err, location, "Creating AES block cipher")
	}

	// Create GCM
	// UNREACHABLE ERROR: NewGCM fails only on a block size other than 16, and aes.NewCipher
	// never returns one. Checked anyway, because the alternative is trusting that forever.
	aesgcm, err := cipher.NewGCM(block)

	if err != nil {
		return derp.Wrap(err, location, "Generating GCM cipher")
	}

	// Encrypt all plaintext values in the vault
	for property, value := range vault.plaintext {

		// DEFENSIVE: SetString is the only writer of `plaintext` and it already refuses
		// empty and obscured values, so this cannot fire today. It stays because the cost
		// of a future second writer forgetting that rule is a placeholder sealed as if it
		// were a secret.
		if !isEncryptable(value) {
			continue
		}

		// RULE: every value gets a FRESH nonce, every time it is sealed. AES-GCM is counter
		// mode: reusing a (key, nonce) pair across two values leaks the XOR of their
		// plaintexts, and reusing one across two versions of the same value leaks it against
		// whatever older copy an attacker already holds. A stored nonce is an OUTPUT of
		// sealing and must never become an input to it.
		nonce, err := newNonce(aesgcm)

		if err != nil {
			return derp.Wrap(err, location, "Generating nonce", property)
		}

		ciphertext := aesgcm.Seal(nil, nonce, []byte(value), nil)

		vault.Encrypted[property] = hex.EncodeToString(ciphertext)
		vault.Nonces[property] = hex.EncodeToString(nonce)
	}

	return nil
}

// Decrypt opens the named values in this Vault using the provided key
func (vault Vault) Decrypt(encryptionKey []byte, values ...string) (mapof.String, error) {

	const location = "model.vault.Decrypt"

	result := make(mapof.String, len(values))

	// If there are no encrypted values, then there is nothing to decrypt, so skip
	// cipher setup entirely.  This mirrors the early exit in Encrypt.
	if len(vault.Encrypted) == 0 {
		vault.patchPlaintext(result, values...)
		return result, nil
	}

	// Create AES block cipher
	block, err := aes.NewCipher(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Creating AES block cipher")
	}

	// Create GCM
	// UNREACHABLE ERROR: see the matching note in Encrypt.
	aesgcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, derp.Wrap(err, location, "Generating GCM cipher")
	}

	// Decode ciphertext values
	for property, value := range vault.Encrypted {

		// If values are specified, then only decrypt those values.
		if len(values) > 0 {
			if !slices.Contains(values, property) {
				continue
			}
		}

		// Resolve the nonce for THIS value, falling back to the shared nonce written
		// by the earlier format for records that have not been re-saved since.
		nonce, err := vault.nonceFor(aesgcm, property)

		if err != nil {
			return nil, derp.Wrap(err, location, "Reading nonce for value in vault", property)
		}

		ciphertext, err := hex.DecodeString(value)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid ciphertext in vault", property)
		}

		plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)

		if err != nil {
			return nil, derp.Wrap(err, location, "Decrypting value in vault", property)
		}

		result[property] = string(plaintext)
	}

	// Patch plaintext values into the result
	vault.patchPlaintext(result, values...)

	// Success.
	return result, nil
}

// patchPlaintext copies the vault's not-yet-encrypted plaintext values into result,
// honoring the same optional property filter that Decrypt applies.
func (vault Vault) patchPlaintext(result mapof.String, values ...string) {

	// If a property is in the plaintext, then it hasn't been encrypted/saved yet.
	// It is still valid to use, so include it here, if applicable.
	for property, value := range vault.plaintext {

		// If values are specified, then only include those values.
		if len(values) > 0 {
			if !slices.Contains(values, property) {
				continue
			}
		}

		result[property] = value
	}
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

// nonceFor returns the nonce that opens the named value: the one stored beside it, or the
// shared nonce written by the earlier format when this value predates the per-value nonces.
// It never generates one -- decryption is a read, and a missing nonce is a fact about the
// stored record, not something to invent a substitute for.
func (vault Vault) nonceFor(aesgcm cipher.AEAD, property string) ([]byte, error) {

	const location = "model.vault.nonceFor"

	encoded, ok := vault.Nonces[property]

	// LEGACY: fall back to the single nonce that older records share across every value
	if !ok || encoded == "" {
		encoded = vault.Nonce
	}

	if encoded == "" {
		return nil, derp.Internal(location, "Vault has no nonce for this value", property)
	}

	nonce, err := hex.DecodeString(encoded)

	if err != nil {
		return nil, derp.Wrap(err, location, "Invalid nonce in vault", property)
	}

	// RULE: GCM PANICS on a nonce of the wrong size rather than returning an error, so a
	// truncated or corrupted value read from the database would crash the request that
	// touched it.  Check the length here, where it can be reported instead.
	if len(nonce) != aesgcm.NonceSize() {
		return nil, derp.Internal(location, "Nonce in vault is the wrong length", property, len(nonce), aesgcm.NonceSize())
	}

	return nonce, nil
}

// newNonce returns a fresh, cryptographically random nonce sized for the provided cipher
func newNonce(aesgcm cipher.AEAD) ([]byte, error) {

	const location = "model.vault.newNonce"

	nonce := make([]byte, aesgcm.NonceSize())

	// UNREACHABLE ERROR: crypto/rand.Reader failing is a process-level fault, not a
	// condition this package can recover from -- but a silently short nonce would be far
	// worse than an error, so it is checked.
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, derp.Wrap(err, location, "Generating nonce")
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
