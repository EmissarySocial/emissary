package random

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateString exercises GenerateString and logs the value it produces
func TestGenerateString(t *testing.T) {
	t.Log(GenerateString(32))
}

// TestBase64URLEncode verifies that encoded values are URL-safe and unpadded
func TestBase64URLEncode(t *testing.T) {
	require.Equal(t, Base64URLEncode([]byte("hello+world")), "aGVsbG8rd29ybGQ")
	require.Equal(t, Base64URLEncode([]byte("1234567890123")), "MTIzNDU2Nzg5MDEyMw")
}
