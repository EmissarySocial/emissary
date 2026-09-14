package ascache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLookupKeyHost pins which lookup keys carry a host, and which host that is
func TestLookupKeyHost(t *testing.T) {

	table := []struct {
		key  string
		host string
	}{
		{"https://good.example/@bob", "good.example"},
		{"http://Good.Example:8080/users/bob?x=1#main-key", "good.example"},
		{"@bob@good.example", "good.example"},
		{"@bob@Good.Example", "good.example"},
		{"bob@good.example", "good.example"},
		{"acct:bob@good.example", "good.example"},
		{"#hashtag", ""},
		{"@alice.bsky.social", ""},
		{"@", ""},
		{"@bob@", ""},
		{"", ""},
		{"not a key at all", ""},
	}

	for _, item := range table {
		require.Equal(t, item.host, lookupKeyHost(item.key), "key %q", item.key)
	}
}

// TestSameHost pins the aliasing rule: a key is an alias only on the document's own host
func TestSameHost(t *testing.T) {

	const documentID = "https://good.example/@bob"

	table := []struct {
		key    string
		result bool
	}{
		{"https://good.example/users/bob", true},
		{"https://GOOD.example/@bob", true},
		{"@bob@good.example", true},
		{"@alice@good.example", true},
		{"@alice@evil.example", false},
		{"https://evil.example/@bob", false},
		{"#hashtag", false},
		{"@alice.bsky.social", false},
		{"", false},
	}

	for _, item := range table {
		require.Equal(t, item.result, sameHost(item.key, documentID), "key %q", item.key)
	}

	// A document with no host vouches for nothing either
	require.False(t, sameHost("https://good.example/@bob", ""))
	require.False(t, sameHost("#hashtag", "#hashtag"))
}
