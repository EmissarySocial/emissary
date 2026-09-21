package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteStatusID_RoundTrip(t *testing.T) {

	postURL := "https://infosec.exchange/users/vmcall/statuses/117304046356952753"
	encoded := EncodeRemoteStatusID(postURL)

	require.NotContains(t, encoded, "/")

	decoded, ok := DecodeRemoteStatusID(encoded)
	require.True(t, ok)
	require.Equal(t, postURL, decoded)
}

func TestRemoteStatusID_RejectsOtherIDs(t *testing.T) {

	for _, id := range []string{"", "6ab03bc05a17f37157bf5d2f", "u_" + EncodeRemoteStatusID("https://example.com/a")[2:], "p_!!!", "p_" + "bm90LWEtdXJs"} {
		_, ok := DecodeRemoteStatusID(id)
		require.False(t, ok, id)
	}
}
