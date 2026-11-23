package server

import (
	"testing"

	"github.com/benpate/hannibal/delivery"
	"github.com/stretchr/testify/require"
)

// TestHostOf verifies that a host is named in lower case with no port, and that a value with no
// host names nothing
func TestHostOf(t *testing.T) {
	require.Equal(t, "remote.example", hostOf("https://remote.example/@alice"))
	require.Equal(t, "remote.example", hostOf("HTTPS://Remote.EXAMPLE/@alice"))
	require.Equal(t, "remote.example", hostOf("https://remote.example:8443/@alice"))
	require.Equal(t, "remote.example", hostOf("https://user:secret@remote.example/@alice"))
	require.Empty(t, hostOf(""))
	require.Empty(t, hostOf("not a url"))
	require.Empty(t, hostOf("mailto:alice@remote.example"))
	require.Empty(t, hostOf("://bad"))
}

// TestHostOf_MatchesDeliveryKey verifies that an inbound request revives the same record that
// hannibal's delivery pipeline reads, however the URL spells the host
func TestHostOf_MatchesDeliveryKey(t *testing.T) {

	for _, value := range []string{
		"https://remote.example/@alice",
		"https://REMOTE.example:443/@alice",
		"https://remote.example:8443/inbox",
		"http://Remote.Example/users/alice#main-key",
		"mailto:alice@remote.example",
		"not a url",
		"",
	} {
		require.Equal(t, delivery.HostOf(value), hostOf(value), value)
	}
}
