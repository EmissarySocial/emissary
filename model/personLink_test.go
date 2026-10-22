package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestPersonLink verifies that every PersonLink property round-trips through the schema
func TestPersonLink(t *testing.T) {

	s := schema.New(PersonLinkSchema())
	response := NewPersonLink()

	tests := []tableTestItem{
		{"userId", "000000000000000000000001", nil},
		{"name", "John Connor", nil},
		{"username", "@john@connor.social", nil},
		{"profileUrl", "https://john.connor.mil", nil},
		{"inboxUrl", "https://john.connor.mil/inbox", nil},
		{"emailAddress", "john.connor@mil", nil},
		{"iconUrl", "https://john.connor.mil/image", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}

// Username is stored with a leading "@" ("e.g. @user@domain.social", per the field's
// own comment, and TestPersonLink's own fixture) -- LocalUsername must strip it, not
// split on it as though it were the user/domain separator.
func TestPersonLink_LocalUsername(t *testing.T) {

	tests := map[string]string{
		"@devInTheVoid@mastodon.social": "devInTheVoid",
		"devInTheVoid@mastodon.social":  "devInTheVoid",
		"@localonly":                    "localonly",
		"localonly":                     "localonly",
		"":                              "",
	}

	for username, expected := range tests {
		person := PersonLink{Username: username}
		require.Equal(t, expected, person.LocalUsername(), username)
	}
}

func TestPersonLink_Toot_RemoteAccount(t *testing.T) {

	person := PersonLink{
		Name:       "devInTheVoid",
		Username:   "@devInTheVoid@mastodon.social",
		ProfileURL: "https://mastodon.social/ap/users/117111409165181205",
		IconURL:    "https://mastodon.social/icon.png",
	}

	account := person.Toot()

	// The same ID scheme GetAccount_Lookup hands out for the same account -- not
	// the raw profile URL (which contains slashes and breaks a route if echoed
	// back as an :id).
	require.Equal(t, EncodeRemoteAccountID(person.ProfileURL), account.ID)
	require.NotContains(t, account.ID, "/")

	require.Equal(t, "devInTheVoid", account.Username)
	require.Equal(t, "devInTheVoid@mastodon.social", account.Acct)
	require.NotContains(t, account.Acct, "@mastodon.social@") // no double-qualification
	require.False(t, account.Acct[0] == '@')                  // never a leading "@"
}

func TestPersonLink_Toot_LocalAccount(t *testing.T) {

	userID, _ := primitive.ObjectIDFromHex("000000000000000000000001")

	person := PersonLink{
		UserID:     userID,
		Username:   "localuser",
		ProfileURL: "https://example.com/@localuser",
	}

	account := person.Toot()

	// A local account keeps its short hex UserID -- unaffected by the remote-ID fix
	require.Equal(t, userID.Hex(), account.ID)
}
