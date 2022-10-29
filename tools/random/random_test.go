package random

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateString(t *testing.T) {
	t.Log(GenerateString(32))
}

func TestBase64URLEncode(t *testing.T) {
	require.Equal(t, Base64URLEncode([]byte("hello+world")), "aGVsbG8rd29ybGQ")
	require.Equal(t, Base64URLEncode([]byte("1234567890123")), "MTIzNDU2Nzg5MDEyMw")
}
