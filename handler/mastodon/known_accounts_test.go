package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestAccountFromFollowing confirms a Following record becomes a usable basic account
func TestAccountFromFollowing(t *testing.T) {

	following := model.NewFollowing()
	following.Label = "devInTheVoid"
	following.Username = "@devInTheVoid@mastodon.social"
	following.ProfileURL = "https://mastodon.social/ap/users/117111409165181205"
	following.IconURL = "https://mastodon.social/avatar.png"

	account := accountFromFollowing(&following)

	require.Equal(t, "devInTheVoid", account.Username)
	require.Equal(t, "devInTheVoid@mastodon.social", account.Acct)
	require.Equal(t, "devInTheVoid", account.DisplayName)
	require.Equal(t, following.ProfileURL, account.URL)
	require.Equal(t, following.IconURL, account.Avatar)
	require.Equal(t, model.EncodeRemoteAccountID(following.ProfileURL), account.ID, "the same ID every other surface uses for this account")
}
