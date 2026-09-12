package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestExport_UserOmitsCredentials is the privacy property for the data-portability export.
//
// RULE: /@:userId/export/... is reachable with a third-party OAuth grant, so anything
// json.Marshal writes for a User is handed to that application.  The password hash is
// offline-crackable and the PasswordReset AuthCode is a live account-takeover token, so
// neither may ever appear in the marshaled document.
func TestExport_UserOmitsCredentials(t *testing.T) {

	user := model.NewUser()
	user.Password = "$2a$10$CANARY-HASHED-PASSWORD"
	user.PasswordReset = model.NewPasswordReset(model.PasswordResetDurationReset)
	user.DisplayName = "Example Person"

	document, err := json.Marshal(user)
	require.Nil(t, err)

	require.NotContains(t, string(document), user.Password, "export must not carry the password hash")
	require.NotContains(t, string(document), user.PasswordReset.AuthCode, "export must not carry a live password-reset code")
	require.NotContains(t, string(document), `"Password"`)
	require.NotContains(t, string(document), `"PasswordReset"`)

	// The export must still carry the profile data it exists to move
	require.Contains(t, string(document), "Example Person")
}

// TestExport_FollowingOmitsSecret pins the same property for a Following, whose Secret
// authenticates Emissary to the remote feed.
func TestExport_FollowingOmitsSecret(t *testing.T) {

	following := model.NewFollowing()
	following.Secret = "CANARY-FEED-SECRET"
	following.Label = "Example Feed"

	document, err := json.Marshal(following)
	require.Nil(t, err)

	require.NotContains(t, string(document), following.Secret, "export must not carry the feed secret")
	require.NotContains(t, string(document), `"Secret"`)
	require.Contains(t, string(document), "Example Feed")
}

// TestExport_UserImportIgnoresCredentials proves the round trip is not silently broken:
// service.User.Import maps IDs only, so dropping the credential fields costs it nothing.
func TestExport_UserImportIgnoresCredentials(t *testing.T) {

	user := model.NewUser()
	user.Password = "$2a$10$CANARY-HASHED-PASSWORD"

	document, err := json.Marshal(user)
	require.Nil(t, err)

	imported := model.NewUser()
	require.Nil(t, json.Unmarshal(document, &imported))

	require.Equal(t, "", imported.Password)
	require.Equal(t, user.UserID, imported.UserID, "the ID mapping that Import actually reads still survives")
	require.False(t, strings.Contains(string(document), "CANARY"))
}
