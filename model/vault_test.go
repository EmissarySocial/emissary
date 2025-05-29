package model

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVault(t *testing.T) {

	encryptionKey, _ := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")

	vault := NewVault()

	// Set Values
	vault.SetString("numbers", "1234567890")
	vault.SetString("letters", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	vault.SetString("symbols", "!@#$%^&*()")
	vault.SetString("empty", "")

	// Encrypt/Save Values
	err := vault.Encrypt(encryptionKey)
	require.Nil(t, err)

	// Retrieve Values
	numbers, ok := vault.GetStringOK("numbers")
	require.True(t, ok)
	require.Equal(t, VaultObscuredValue, numbers)

	letters, ok := vault.GetStringOK("letters")
	require.True(t, ok)
	require.Equal(t, VaultObscuredValue, letters)

	symbols, ok := vault.GetStringOK("symbols")
	require.True(t, ok)
	require.Equal(t, VaultObscuredValue, symbols)

	empty, ok := vault.GetStringOK("empty")
	require.False(t, ok)
	require.Equal(t, "", empty)

	missing, ok := vault.GetStringOK("missing")
	require.False(t, ok)
	require.Equal(t, "", missing)

	decrypted, err := vault.Decrypt(encryptionKey)
	require.Nil(t, err)

	require.Equal(t, "1234567890", decrypted["numbers"])
	require.Equal(t, "ABCDEFGHIJKLMNOPQRSTUVWXYZ", decrypted["letters"])
	require.Equal(t, "!@#$%^&*()", decrypted["symbols"])
}
