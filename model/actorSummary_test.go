package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestActorSummary_UsernameOrID covers the two branches of UsernameOrID: a WebFinger-style handle
// built from PreferredUsername + the host parsed out of ID, or a fallback to the raw ID.
func TestActorSummary_UsernameOrID(t *testing.T) {

	// With a preferred username, build "@user@host" using the hostname parsed from the ID URL.
	withUsername := ActorSummary{
		ID:                "https://example.com/@sarah",
		PreferredUsername: "sarah",
	}
	require.Equal(t, "@sarah@example.com", withUsername.UsernameOrID())

	// Subdomain hosts are preserved.
	subdomain := ActorSummary{
		ID:                "https://social.example.com/users/bob",
		PreferredUsername: "bob",
	}
	require.Equal(t, "@bob@social.example.com", subdomain.UsernameOrID())

	// Without a preferred username, fall back to the raw ID.
	noUsername := ActorSummary{
		ID: "https://example.com/12345",
	}
	require.Equal(t, "https://example.com/12345", noUsername.UsernameOrID())
}

// TestActorSummary_Username confirms the short alias returns PreferredUsername verbatim.
func TestActorSummary_Username(t *testing.T) {
	actor := ActorSummary{PreferredUsername: "sarah"}
	require.Equal(t, "sarah", actor.Username())
}
