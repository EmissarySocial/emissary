package model

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzVault_RoundTrip asserts the property that matters for a sealed value: whatever goes in
// comes back out, byte for byte, for any property name and any value. Values that the Vault
// deliberately refuses to store -- empty strings and the obscured placeholder -- are asserted
// to be absent rather than round-tripped, because storing them is the bug, not the contract.
func FuzzVault_RoundTrip(f *testing.F) {

	f.Add("key", "value")
	f.Add("", "")
	f.Add("key", "")
	f.Add("key", VaultObscuredValue)
	f.Add(VaultObscuredValue, "value")
	f.Add("unicode", "こんにちは 🌍")
	f.Add("invalid-utf8", "\xff\xfe\xfd")
	f.Add("nul", "before\x00after")
	f.Add("newline", "line\nline")
	f.Add("long", strings.Repeat("x", 4096))
	f.Add("hex-shaped", "6368616e6765")

	encryptionKey, err := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, name string, value string) {

		vault := NewVault()
		vault.SetString(name, value)

		require.NotPanics(t, func() {
			require.NoError(t, vault.Encrypt(encryptionKey))
		})

		decrypted, err := vault.Decrypt(encryptionKey)
		require.NoError(t, err)

		// A value the Vault refuses to store must not appear anywhere
		if !isEncryptable(value) {
			require.NotContains(t, vault.Encrypted, name)
			require.NotContains(t, vault.Nonces, name)
			return
		}

		require.Equal(t, value, decrypted[name])

		// Every sealed value carries its own correctly-sized nonce
		nonce, err := hex.DecodeString(vault.Nonces[name])
		require.NoError(t, err)
		require.Len(t, nonce, 12)
	})
}

// FuzzVault_RoundTripPair asserts the same property with TWO values present, which is the
// case the shared nonce used to break. It also asserts the invariant directly: two values
// sealed together never share a nonce.
func FuzzVault_RoundTripPair(f *testing.F) {

	f.Add("a", "value-a", "b", "value-b")
	f.Add("same", "identical", "other", "identical")
	f.Add("a", "", "b", "value-b")
	f.Add("collide", "x", "collide", "y")
	f.Add("pk", "pk_live_AAAAAAAA", "rk", "rk_live_BBBBBBBB")

	encryptionKey, err := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, firstName string, firstValue string, secondName string, secondValue string) {

		vault := NewVault()
		vault.SetString(firstName, firstValue)
		vault.SetString(secondName, secondValue)

		require.NoError(t, vault.Encrypt(encryptionKey))

		decrypted, err := vault.Decrypt(encryptionKey)
		require.NoError(t, err)

		// Whichever writes survived must round-trip. A repeated name is last-write-wins,
		// so only the surviving value is asserted.
		for name := range vault.Encrypted {
			require.Contains(t, decrypted, name)
		}

		if isEncryptable(firstValue) && (firstName != secondName) {
			require.Equal(t, firstValue, decrypted[firstName])
		}

		if isEncryptable(secondValue) {
			require.Equal(t, secondValue, decrypted[secondName])
		}

		// THE invariant: no two sealed values share a nonce
		seen := make(map[string]bool, len(vault.Nonces))

		for property, nonce := range vault.Nonces {
			require.Falsef(t, seen[nonce], "nonce reused for %q", property)
			seen[nonce] = true
		}
	})
}

// FuzzVault_DecryptCorrupt is the target that earns its keep. Every field it fuzzes is read
// straight off the database, so every one of them can be arbitrary bytes -- and GCM PANICS
// rather than errors on a nonce of the wrong length, which is exactly the kind of crash a
// hand-written table of cases misses. The property is simple: Decrypt returns an error or a
// result, and never panics.
func FuzzVault_DecryptCorrupt(f *testing.F) {

	f.Add("", "", "")
	f.Add("abc", "abc", "abc")
	f.Add("zz", "zz", "zz")
	f.Add(strings.Repeat("ab", 32), strings.Repeat("ab", 12), "")
	f.Add(strings.Repeat("ab", 32), "", strings.Repeat("ab", 12))
	f.Add(strings.Repeat("ab", 32), strings.Repeat("ab", 11), "")
	f.Add(strings.Repeat("ab", 32), strings.Repeat("ab", 13), "")
	f.Add(strings.Repeat("00", 16), strings.Repeat("00", 12), "")
	f.Add("ff", "ff", "ff")

	encryptionKey, err := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, ciphertext string, nonce string, legacyNonce string) {

		vault := NewVault()
		vault.Encrypted["key"] = ciphertext
		vault.Nonces["key"] = nonce
		vault.Nonce = legacyNonce

		var decrypted map[string]string
		var decryptError error

		require.NotPanics(t, func() {
			result, err := vault.Decrypt(encryptionKey)
			decryptError = err

			if err == nil {
				decrypted = result
			}
		})

		// Error XOR result: a successful decrypt must actually produce the value
		if decryptError == nil {
			require.Contains(t, decrypted, "key")
		}
	})
}

// FuzzVault_KeyLength asserts that a key of any length is handled by the cipher's own
// validation rather than by a panic somewhere downstream
func FuzzVault_KeyLength(f *testing.F) {

	f.Add([]byte(nil))
	f.Add([]byte{})
	f.Add(make([]byte, 15))
	f.Add(make([]byte, 16))
	f.Add(make([]byte, 24))
	f.Add(make([]byte, 32))
	f.Add(make([]byte, 33))
	f.Add(make([]byte, 1024))

	f.Fuzz(func(t *testing.T, encryptionKey []byte) {

		// Sealing with an arbitrary key either works or errors, but never panics
		vault := NewVault()
		vault.SetString("key", "value")

		var encryptError error

		require.NotPanics(t, func() {
			encryptError = vault.Encrypt(encryptionKey)
		})

		// AES accepts only 16, 24, and 32-byte keys
		switch len(encryptionKey) {

		case 16, 24, 32:
			require.NoError(t, encryptError)

			decrypted, err := vault.Decrypt(encryptionKey)
			require.NoError(t, err)
			require.Equal(t, "value", decrypted["key"])

		default:
			require.Error(t, encryptError)
		}
	})
}
