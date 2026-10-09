package model

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Nonce Freshness
 *
 * The invariant this project exists to establish. These are the tests that
 * fail if someone reintroduces a stored-and-reused nonce -- which is the
 * dangerous regression, because it leaks silently while every round-trip
 * test still passes.
 ******************************************/

// TestVault_NonceIsPerValue confirms that two values in one vault never share a nonce
func TestVault_NonceIsPerValue(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("first", "value-one")
	vault.SetString("second", "value-two")
	vault.SetString("third", "value-three")

	require.NoError(t, vault.Encrypt(encryptionKey))

	require.Len(t, vault.Nonces, 3)

	seen := make(map[string]string, 3)

	for property, nonce := range vault.Nonces {
		require.NotEmpty(t, nonce, property)

		if previous, duplicate := seen[nonce]; duplicate {
			require.Failf(t, "nonce reused", "%q and %q share the nonce %s", previous, property, nonce)
		}

		seen[nonce] = property
	}
}

// TestVault_NonceIsPerEncryption confirms the harder half of the invariant: sealing the
// SAME property a second time produces a new nonce, so an attacker holding an older copy
// of the record cannot compare the two ciphertexts.
func TestVault_NonceIsPerEncryption(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("key", "the-same-value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	firstNonce := vault.Nonces["key"]
	firstCiphertext := vault.Encrypted["key"]

	// Seal the identical value again
	require.NoError(t, vault.Encrypt(encryptionKey))

	require.NotEqual(t, firstNonce, vault.Nonces["key"], "re-encryption must mint a new nonce")
	require.NotEqual(t, firstCiphertext, vault.Encrypted["key"], "an identical plaintext must not produce identical ciphertext")

	// ...and it still opens
	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "the-same-value", decrypted["key"])
}

// TestVault_NonceNeverRepeatsAcrossManySaves runs enough encryptions that a stored-and-reused
// nonce could not survive by luck
func TestVault_NonceNeverRepeatsAcrossManySaves(t *testing.T) {

	encryptionKey := testEncryptionKey(t)
	seen := make(map[string]bool, 200)

	vault := NewVault()
	vault.SetString("alpha", "value-alpha")
	vault.SetString("beta", "value-beta")

	for i := 0; i < 100; i++ {

		require.NoError(t, vault.Encrypt(encryptionKey))

		for property, nonce := range vault.Nonces {
			require.Falsef(t, seen[nonce], "nonce %s reused for %q on iteration %d", nonce, property, i)
			seen[nonce] = true
		}
	}

	require.Len(t, seen, 200)
}

// TestVault_EncryptNeverWritesLegacyNonce confirms D2: the shared nonce field is read-only now
func TestVault_EncryptNeverWritesLegacyNonce(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("key", "value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	require.Empty(t, vault.Nonce, "Encrypt must not write the deprecated shared nonce")
	require.NotEmpty(t, vault.Nonces["key"])
}

// TestVault_NonceLength confirms that generated nonces are the size GCM requires.
// A nonce of any other length would panic inside Open rather than fail cleanly.
func TestVault_NonceLength(t *testing.T) {

	encryptionKey := testEncryptionKey(t)
	aesgcm := testGCM(t, encryptionKey)

	vault := NewVault()
	vault.SetString("key", "value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	nonce, err := hex.DecodeString(vault.Nonces["key"])
	require.NoError(t, err)
	require.Len(t, nonce, aesgcm.NonceSize())
}

/******************************************
 * Round Trip
 ******************************************/

// TestVault_RoundTrip covers the ordinary cases, including values that are awkward to encode
func TestVault_RoundTrip(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	values := mapof.String{
		"numbers":  "1234567890",
		"letters":  "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"symbols":  "!@#$%^&*()",
		"unicode":  "こんにちは 🌍 café",
		"newlines": "line one\nline two\r\nline three",
		"long":     strings.Repeat("x", 8192),
		"onechar":  "a",
		"spaces":   "   ",
	}

	vault := NewVault()

	for name, value := range values {
		vault.SetString(name, value)
	}

	require.NoError(t, vault.Encrypt(encryptionKey))

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)

	for name, value := range values {
		require.Equal(t, value, decrypted[name], "property %q did not survive the round trip", name)
	}
}

// TestVault_RoundTripFilter confirms that the property filter selects exactly what was asked for
func TestVault_RoundTripFilter(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("wanted", "yes")
	vault.SetString("unwanted", "no")
	require.NoError(t, vault.Encrypt(encryptionKey))

	decrypted, err := vault.Decrypt(encryptionKey, "wanted")
	require.NoError(t, err)
	require.Equal(t, "yes", decrypted["wanted"])
	require.NotContains(t, decrypted, "unwanted")

	// A filter naming nothing that exists returns nothing, and is not an error
	decrypted, err = vault.Decrypt(encryptionKey, "absent")
	require.NoError(t, err)
	require.Empty(t, decrypted)
}

/******************************************
 * Legacy Compatibility
 ******************************************/

// TestVault_LegacySingleValue confirms that a record written by the original format still opens
func TestVault_LegacySingleValue(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := legacyVault(t, encryptionKey, mapof.String{"only": "legacy-value"})

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "legacy-value", decrypted["only"])
}

// TestVault_LegacyMultipleValues confirms the shared-nonce fallback applies to every value
func TestVault_LegacyMultipleValues(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := legacyVault(t, encryptionKey, mapof.String{
		"publishableKey": "pk_live_ABCDEFGHIJKLMNOP",
		"restrictedKey":  "rk_live_QRSTUVWXYZ012345",
		"webhookSecret":  "whsec_6789abcdefghijkl",
	})

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)

	require.Equal(t, "pk_live_ABCDEFGHIJKLMNOP", decrypted["publishableKey"])
	require.Equal(t, "rk_live_QRSTUVWXYZ012345", decrypted["restrictedKey"])
	require.Equal(t, "whsec_6789abcdefghijkl", decrypted["webhookSecret"])
}

// TestVault_MixedFormat confirms a vault that is half-migrated: one value still relies on the
// shared nonce while another carries its own. This is the state every existing record enters
// the moment it is next saved, so it must not be a special case.
func TestVault_MixedFormat(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	// A record already on disk, in the legacy format
	vault := legacyVault(t, encryptionKey, mapof.String{"old": "old-value"})

	// The owner adds a new secret and saves
	vault.SetString("new", "new-value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	require.NotEmpty(t, vault.Nonce, "the legacy nonce must remain readable for the untouched value")
	require.NotEmpty(t, vault.Nonces["new"], "the new value must carry its own nonce")
	require.NotContains(t, vault.Nonces, "old", "an untouched value is not re-sealed")

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "old-value", decrypted["old"])
	require.Equal(t, "new-value", decrypted["new"])
}

// TestVault_MigratesOnRewrite confirms that re-sealing a legacy value moves it onto its own
// nonce, and that the result no longer depends on the shared one
func TestVault_MigratesOnRewrite(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := legacyVault(t, encryptionKey, mapof.String{"key": "original"})

	// Rewriting the value is what migrates it
	vault.SetString("key", "updated")
	require.NoError(t, vault.Encrypt(encryptionKey))

	require.NotEmpty(t, vault.Nonces["key"])

	// Prove independence from the legacy nonce by removing it entirely
	vault.Nonce = ""

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "updated", decrypted["key"])
}

// TestVault_PerValueNonceWins confirms precedence: when both nonces are present, the per-value
// one is used. A reader that preferred the shared nonce would fail authentication on migrated data.
func TestVault_PerValueNonceWins(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("key", "value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	// Plant a plausible but wrong shared nonce alongside the correct per-value one
	vault.Nonce = strings.Repeat("ab", 12)

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "value", decrypted["key"])
}

/******************************************
 * Tamper Detection
 ******************************************/

// TestVault_TamperDetection confirms that altering stored bytes yields an error, never a
// plaintext. GCM authenticates, and these assert that the authentication is actually reaching us.
func TestVault_TamperDetection(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	newSealedVault := func() Vault {
		vault := NewVault()
		vault.SetString("key", "sensitive-value")
		require.NoError(t, vault.Encrypt(encryptionKey))
		return vault
	}

	// Flip one bit of the ciphertext
	{
		vault := newSealedVault()
		raw, err := hex.DecodeString(vault.Encrypted["key"])
		require.NoError(t, err)
		raw[0] ^= 0x01
		vault.Encrypted["key"] = hex.EncodeToString(raw)

		_, err = vault.Decrypt(encryptionKey)
		require.Error(t, err)
	}

	// Flip one bit of the authentication tag (the last byte)
	{
		vault := newSealedVault()
		raw, err := hex.DecodeString(vault.Encrypted["key"])
		require.NoError(t, err)
		raw[len(raw)-1] ^= 0x01
		vault.Encrypted["key"] = hex.EncodeToString(raw)

		_, err = vault.Decrypt(encryptionKey)
		require.Error(t, err)
	}

	// Flip one bit of the nonce
	{
		vault := newSealedVault()
		raw, err := hex.DecodeString(vault.Nonces["key"])
		require.NoError(t, err)
		raw[0] ^= 0x01
		vault.Nonces["key"] = hex.EncodeToString(raw)

		_, err = vault.Decrypt(encryptionKey)
		require.Error(t, err)
	}

	// Swap two values' ciphertexts, which their separate nonces now make detectable
	{
		vault := NewVault()
		vault.SetString("first", "first-value")
		vault.SetString("second", "second-value")
		require.NoError(t, vault.Encrypt(encryptionKey))

		vault.Encrypted["first"], vault.Encrypted["second"] = vault.Encrypted["second"], vault.Encrypted["first"]

		_, err := vault.Decrypt(encryptionKey)
		require.Error(t, err, "a value moved to another property must not open")
	}
}

/******************************************
 * Corrupt Stored State
 *
 * Everything here arrives from the database, so everything here can be wrong.
 * None of it may panic.
 ******************************************/

// TestVault_CorruptNonce covers every shape of broken nonce, including the one that
// panics inside GCM if it is not caught first
func TestVault_CorruptNonce(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	table := map[string]string{
		"empty":          "",
		"odd length":     "abc",
		"not hex":        "zzzzzzzzzzzzzzzzzzzzzzzz",
		"one byte short": strings.Repeat("ab", 11),
		"one byte long":  strings.Repeat("ab", 13),
		"far too long":   strings.Repeat("ab", 64),
		"single byte":    "ff",
	}

	for name, nonce := range table {

		vault := NewVault()
		vault.SetString("key", "value")
		require.NoError(t, vault.Encrypt(encryptionKey))

		vault.Nonces["key"] = nonce

		// An empty per-value nonce falls back to the shared nonce, which is also absent here
		require.NotPanics(t, func() {
			_, err := vault.Decrypt(encryptionKey)
			require.Error(t, err, "case %q should error", name)
		}, "case %q must not panic", name)
	}
}

// TestVault_MissingNonce confirms that ciphertext with no nonce anywhere is a clear error
func TestVault_MissingNonce(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("key", "value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	delete(vault.Nonces, "key")
	vault.Nonce = ""

	_, err := vault.Decrypt(encryptionKey)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonce")
}

// TestVault_CorruptCiphertext covers broken ciphertext, including values too short to
// contain a GCM authentication tag
func TestVault_CorruptCiphertext(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	table := map[string]string{
		"empty":        "",
		"odd length":   "abc",
		"not hex":      "nothexatall",
		"too short":    "ab",
		"tag only":     strings.Repeat("00", 16),
		"one byte":     "ff",
		"long garbage": strings.Repeat("ab", 1024),
	}

	for name, ciphertext := range table {

		vault := NewVault()
		vault.SetString("key", "value")
		require.NoError(t, vault.Encrypt(encryptionKey))

		vault.Encrypted["key"] = ciphertext

		require.NotPanics(t, func() {
			_, err := vault.Decrypt(encryptionKey)
			require.Error(t, err, "case %q should error", name)
		}, "case %q must not panic", name)
	}
}

// TestVault_OrphanNonce confirms that a nonce with no matching ciphertext is simply ignored.
// Decrypt iterates ciphertexts, so an orphan is inert -- this pins that it stays inert.
func TestVault_OrphanNonce(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("key", "value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	vault.Nonces["ghost"] = strings.Repeat("ab", 12)

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "value", decrypted["key"])
	require.NotContains(t, decrypted, "ghost")
}

// TestVault_NilMaps confirms that a zero-value Vault -- which is what a BSON decode of an
// absent field produces -- neither panics nor misbehaves
func TestVault_NilMaps(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	// A zero Vault decrypts to nothing
	var zero Vault
	decrypted, err := zero.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Empty(t, decrypted)

	// ...and reports no values
	require.False(t, zero.HasString("anything"))
	value, ok := zero.GetStringOK("anything")
	require.False(t, ok)
	require.Empty(t, value)

	// ...and can be written to and sealed, allocating as it goes
	require.True(t, zero.SetString("key", "value"))
	require.NoError(t, zero.Encrypt(encryptionKey))
	require.NotEmpty(t, zero.Nonces["key"])

	decrypted, err = zero.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "value", decrypted["key"])
}

/******************************************
 * Key Handling
 ******************************************/

// TestVault_WrongKey confirms that the wrong key fails authentication rather than
// returning garbage
func TestVault_WrongKey(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	wrongKey, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	vault := NewVault()
	vault.SetString("key", "value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	_, err = vault.Decrypt(wrongKey)
	require.Error(t, err)
}

// TestVault_InvalidKeyLength confirms that AES rejects a key of the wrong size, on both paths
func TestVault_InvalidKeyLength(t *testing.T) {

	table := [][]byte{
		nil,
		{},
		[]byte("too-short"),
		make([]byte, 31),
		make([]byte, 33),
	}

	for _, key := range table {

		// Encrypting with a bad key fails
		vault := NewVault()
		vault.SetString("key", "value")
		require.Error(t, vault.Encrypt(key), "key length %d", len(key))

		// Decrypting a populated vault with a bad key fails
		sealed := NewVault()
		sealed.SetString("key", "value")
		require.NoError(t, sealed.Encrypt(testEncryptionKey(t)))

		_, err := sealed.Decrypt(key)
		require.Error(t, err, "key length %d", len(key))
	}
}

// TestVault_EmptyVaultIgnoresKey pins the existing early exits: a vault with nothing to do
// never sets up a cipher, so an unusable key is not an error
func TestVault_EmptyVaultIgnoresKey(t *testing.T) {

	vault := NewVault()

	require.NoError(t, vault.Encrypt(nil))

	decrypted, err := vault.Decrypt(nil)
	require.NoError(t, err)
	require.Empty(t, decrypted)
}

/******************************************
 * Preserved Behavior
 *
 * Everything below worked before this change and must keep working.
 ******************************************/

// TestVault_ObscuredValueIsNotStored confirms the round-trip guard that lets a settings form
// redisplay a secret without erasing it. Two scenarios, because they take different paths:
// a record freshly loaded from the database holds no plaintext at all, while one still in
// memory holds the value it just sealed.
func TestVault_ObscuredValueIsNotStored(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	// SCENARIO 1: a record loaded from the database, with no plaintext in memory.
	// The obscured placeholder must not reach the vault, and with nothing encryptable
	// left, Encrypt exits before the cipher and leaves the stored value untouched.
	{
		stored := NewVault()
		stored.SetString("key", "real-secret")
		require.NoError(t, stored.Encrypt(encryptionKey))

		loaded := NewVault()
		loaded.Encrypted["key"] = stored.Encrypted["key"]
		loaded.Nonces["key"] = stored.Nonces["key"]

		require.True(t, loaded.SetString("key", VaultObscuredValue))
		require.NoError(t, loaded.Encrypt(encryptionKey))

		require.Equal(t, stored.Encrypted["key"], loaded.Encrypted["key"], "an untouched value must not be re-sealed")
		require.Equal(t, stored.Nonces["key"], loaded.Nonces["key"])

		decrypted, err := loaded.Decrypt(encryptionKey)
		require.NoError(t, err)
		require.Equal(t, "real-secret", decrypted["key"], "the placeholder must never become the stored value")
	}

	// SCENARIO 2: a vault still holding the plaintext it sealed a moment ago. The
	// placeholder is still refused, so the real value is re-sealed -- with a FRESH nonce,
	// which is why the ciphertext changes even though the secret did not. Under the old
	// shared-nonce format this produced byte-identical ciphertext; producing new bytes
	// on every save is the point of the change, not a regression.
	{
		vault := NewVault()
		vault.SetString("key", "real-secret")
		require.NoError(t, vault.Encrypt(encryptionKey))

		firstCiphertext := vault.Encrypted["key"]
		firstNonce := vault.Nonces["key"]

		require.True(t, vault.SetString("key", VaultObscuredValue))
		require.NoError(t, vault.Encrypt(encryptionKey))

		require.NotEqual(t, firstCiphertext, vault.Encrypted["key"])
		require.NotEqual(t, firstNonce, vault.Nonces["key"])

		decrypted, err := vault.Decrypt(encryptionKey)
		require.NoError(t, err)
		require.Equal(t, "real-secret", decrypted["key"], "the placeholder must never become the stored value")
	}
}

// TestVault_EmptyValueDeletes confirms that clearing a value removes the ciphertext AND its
// nonce, so no orphan outlives the value it described
func TestVault_EmptyValueDeletes(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("key", "value")
	vault.SetString("keeper", "kept")
	require.NoError(t, vault.Encrypt(encryptionKey))

	require.NotEmpty(t, vault.Nonces["key"])

	vault.SetString("key", "")

	require.NotContains(t, vault.Encrypted, "key")
	require.NotContains(t, vault.Nonces, "key", "a deleted value must not leave its nonce behind")
	require.False(t, vault.HasString("key"))

	// The other value is untouched
	require.True(t, vault.HasString("keeper"))

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.NotContains(t, decrypted, "key")
	require.Equal(t, "kept", decrypted["keeper"])
}

// TestVault_PlaintextIsPatchedIn confirms that a value set but not yet sealed is still
// readable, under the same property filter Decrypt applies to ciphertexts
func TestVault_PlaintextIsPatchedIn(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("sealed", "sealed-value")
	require.NoError(t, vault.Encrypt(encryptionKey))

	vault.SetString("pending", "not-saved-yet")

	decrypted, err := vault.Decrypt(encryptionKey)
	require.NoError(t, err)
	require.Equal(t, "sealed-value", decrypted["sealed"])
	require.Equal(t, "not-saved-yet", decrypted["pending"])

	// The filter applies to pending values too
	filtered, err := vault.Decrypt(encryptionKey, "pending")
	require.NoError(t, err)
	require.Equal(t, "not-saved-yet", filtered["pending"])
	require.NotContains(t, filtered, "sealed")
}

// TestVault_GetStringOK confirms that reading never reveals a secret, only its presence
func TestVault_GetStringOK(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	vault := NewVault()
	vault.SetString("sealed", "sealed-value")
	require.NoError(t, vault.Encrypt(encryptionKey))
	vault.SetString("pending", "pending-value")

	for _, name := range []string{"sealed", "pending"} {
		value, ok := vault.GetStringOK(name)
		require.True(t, ok, name)
		require.Equal(t, VaultObscuredValue, value, name)
		require.True(t, vault.HasString(name), name)
	}

	value, ok := vault.GetStringOK("missing")
	require.False(t, ok)
	require.Empty(t, value)
	require.False(t, vault.HasString("missing"))
}

/******************************************
 * The Regression Test
 ******************************************/

// TestVault_NoSharedNonceLeak is the test that fails without this fix. It runs the exact
// attack from TestLegacyVault_SharedNonceLeaks -- recovering one plaintext from another
// using only their ciphertexts -- against a vault sealed by the current code, and asserts
// that it no longer works.
//
// Note what a plain round-trip test would NOT have caught: under the shared nonce, every
// value still encrypted and decrypted perfectly. The leak was invisible to correctness
// tests, which is why this one asserts the absence of a relationship between ciphertexts.
func TestVault_NoSharedNonceLeak(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	const knownPlaintext = "AAAAAAAAAA"
	const secretPlaintext = "hunter2xyz"

	vault := NewVault()
	vault.SetString("known", knownPlaintext)
	vault.SetString("secret", secretPlaintext)
	require.NoError(t, vault.Encrypt(encryptionKey))

	known, err := hex.DecodeString(vault.Encrypted["known"])
	require.NoError(t, err)

	secret, err := hex.DecodeString(vault.Encrypted["secret"])
	require.NoError(t, err)

	// Attempt the recovery that succeeded against the legacy format
	recovered := make([]byte, len(secretPlaintext))

	for i := range recovered {
		recovered[i] = known[i] ^ secret[i] ^ knownPlaintext[i]
	}

	require.NotEqual(t, secretPlaintext, string(recovered),
		"separate nonces must break the keystream relationship between two values")

	// And the nonces really are different, which is why
	require.NotEqual(t, vault.Nonces["known"], vault.Nonces["secret"])
}

// TestVault_NoReEncryptionLeak is the same assertion for the other half of the defect:
// an attacker holding an older copy of the SAME property must not be able to compare it
// against the current one.
func TestVault_NoReEncryptionLeak(t *testing.T) {

	encryptionKey := testEncryptionKey(t)

	const oldPlaintext = "old-value!"
	const newPlaintext = "new-secret"

	vault := NewVault()
	vault.SetString("key", oldPlaintext)
	require.NoError(t, vault.Encrypt(encryptionKey))

	// The attacker's stolen backup
	backup, err := hex.DecodeString(vault.Encrypted["key"])
	require.NoError(t, err)
	backupNonce := vault.Nonces["key"]

	// The owner updates the secret and saves
	vault.SetString("key", newPlaintext)
	require.NoError(t, vault.Encrypt(encryptionKey))

	current, err := hex.DecodeString(vault.Encrypted["key"])
	require.NoError(t, err)

	require.NotEqual(t, backupNonce, vault.Nonces["key"], "an update must not reuse the previous nonce")

	recovered := make([]byte, len(newPlaintext))

	for i := range recovered {
		recovered[i] = backup[i] ^ current[i] ^ oldPlaintext[i]
	}

	require.NotEqual(t, newPlaintext, string(recovered),
		"a fresh nonce on every save must break the relationship to an older ciphertext")
}

// TestVault_ZeroValueEncrypt confirms that sealing a Vault that was never constructed
// through NewVault -- the shape a BSON decode produces when the field is absent -- allocates
// its maps instead of panicking on a nil write
func TestVault_ZeroValueEncrypt(t *testing.T) {

	var zero Vault

	require.NoError(t, zero.Encrypt(testEncryptionKey(t)))

	require.NotNil(t, zero.Encrypted)
	require.NotNil(t, zero.Nonces)
	require.Empty(t, zero.Encrypted)
	require.Empty(t, zero.Nonces)
}
