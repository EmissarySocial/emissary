package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Legacy Format Fixtures
 *
 * These helpers reproduce the ORIGINAL Vault format, which sealed every
 * value in a vault with a single shared nonce stored in `Vault.Nonce`.
 * That format is a DEFECT -- reusing an AES-GCM nonce across two values
 * leaks the XOR of their plaintexts -- and these helpers exist only so
 * that the current reader can be proven backward compatible with data
 * already on disk.
 *
 * Do not copy this as an example of how to seal anything.
 *
 * They are written from raw AES-GCM calls rather than by calling
 * Vault.Encrypt, because Encrypt is the thing under test: a fixture built
 * from it would silently follow it wherever it went.
 ******************************************/

// testEncryptionKey is a fixed 32-byte AES-256 key, so that failures are reproducible
func testEncryptionKey(t *testing.T) []byte {
	t.Helper()

	key, err := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")
	require.NoError(t, err)
	require.Len(t, key, 32)

	return key
}

// testGCM returns an AES-GCM cipher for the provided key
func testGCM(t *testing.T, encryptionKey []byte) cipher.AEAD {
	t.Helper()

	block, err := aes.NewCipher(encryptionKey)
	require.NoError(t, err)

	aesgcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	return aesgcm
}

// legacyVault builds a Vault in the ORIGINAL on-disk format: every value sealed
// with one shared nonce, recorded in the `Nonce` field and nowhere else.
func legacyVault(t *testing.T, encryptionKey []byte, values mapof.String) Vault {
	t.Helper()

	aesgcm := testGCM(t, encryptionKey)

	nonce := make([]byte, aesgcm.NonceSize())
	_, err := io.ReadFull(rand.Reader, nonce)
	require.NoError(t, err)

	return legacyVaultWithNonce(t, encryptionKey, values, nonce)
}

// legacyVaultWithNonce is legacyVault with a caller-supplied nonce, so that a test can
// pin the reuse the original format permitted.
func legacyVaultWithNonce(t *testing.T, encryptionKey []byte, values mapof.String, nonce []byte) Vault {
	t.Helper()

	aesgcm := testGCM(t, encryptionKey)

	vault := NewVault()
	vault.Nonce = hex.EncodeToString(nonce)

	for name, value := range values {
		ciphertext := aesgcm.Seal(nil, nonce, []byte(value), nil)
		vault.Encrypted[name] = hex.EncodeToString(ciphertext)
	}

	return vault
}

/******************************************
 * Characterization Tests
 *
 * These pin the behavior of the legacy format itself, so that the
 * fixtures above are proven to reproduce it before anything relies on them.
 ******************************************/

// TestLegacyVault_Fixture confirms the fixture really is in the legacy shape:
// a shared nonce, no per-value nonces, and readable ciphertext.
func TestLegacyVault_Fixture(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := legacyVault(t, encryptionKey, mapof.String{
		"first":  "value-one",
		"second": "value-two",
	})

	// The legacy format records exactly one nonce, for the whole vault
	require.NotEmpty(t, vault.Nonce)
	require.Empty(t, vault.Nonces, "the legacy format has no per-value nonces")

	require.Len(t, vault.Encrypted, 2)
	require.NotEmpty(t, vault.Encrypted["first"])
	require.NotEmpty(t, vault.Encrypted["second"])
}

// TestLegacyVault_SharedNonceLeaks demonstrates the defect this project fixes, so that
// nobody has to take the reasoning on faith: two values sealed under one nonce satisfy
// ciphertext1 XOR ciphertext2 == plaintext1 XOR plaintext2, which recovers either
// plaintext from the other WITHOUT the encryption key.
func TestLegacyVault_SharedNonceLeaks(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	// Two values of equal length, sealed in one legacy vault
	vault := legacyVault(t, encryptionKey, mapof.String{
		"known":  "AAAAAAAAAA",
		"secret": "hunter2xyz",
	})

	known, err := hex.DecodeString(vault.Encrypted["known"])
	require.NoError(t, err)

	secret, err := hex.DecodeString(vault.Encrypted["secret"])
	require.NoError(t, err)

	// Recover the unknown plaintext using only the two ciphertexts and the known plaintext
	recovered := make([]byte, len("hunter2xyz"))

	for i := range recovered {
		recovered[i] = known[i] ^ secret[i] ^ "AAAAAAAAAA"[i]
	}

	require.Equal(t, "hunter2xyz", string(recovered),
		"the legacy shared nonce leaks one value given another -- this is the defect being fixed")
}

// TestLegacyVault_ReEncryptionReusedTheNonce pins the second half of the defect: under the
// legacy format, updating a value re-sealed it with the SAME stored nonce, so an attacker
// holding an older backup could compare the two ciphertexts.
func TestLegacyVault_ReEncryptionReusedTheNonce(t *testing.T) {

	encryptionKey := testEncryptionKey(t)
	aesgcm := testGCM(t, encryptionKey)

	nonce := make([]byte, aesgcm.NonceSize())
	_, err := io.ReadFull(rand.Reader, nonce)
	require.NoError(t, err)

	before := legacyVaultWithNonce(t, encryptionKey, mapof.String{"key": "old-value"}, nonce)
	after := legacyVaultWithNonce(t, encryptionKey, mapof.String{"key": "new-value"}, nonce)

	require.Equal(t, before.Nonce, after.Nonce)
	require.NotEqual(t, before.Encrypted["key"], after.Encrypted["key"])
}
