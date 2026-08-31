package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestFollowerUnsubscribe_ReachableByRecipient pins the one property the unsubscribe link in
// every email depends on: the person who clicks it can actually reach it.
//
// The link goes to an email follower, who has no account and no session -- so the action cannot
// be gated on a role. It shipped as roles:["self"], which meant only the profile owner could run
// it, and every recipient who clicked "unsubscribe" got a 403. Nothing failed loudly: the mail
// was sent, the header was correct, and the URL was right; it simply refused the one visitor it
// was written for. RFC 8058 one-click makes that worse, because the POST comes from the mail
// provider, which is even less authenticated than the recipient.
//
// Authorization lives in the URL instead. with-follower refuses to load anything without a
// matching followerId+secret pair (Follower.LoadBySecret), and refuses outright when no
// followerId is supplied, so the open role does not open the action.
func TestFollowerUnsubscribe_ReachableByRecipient(t *testing.T) {

	templateService := loadEmbeddedTemplates(t)

	template, exists := templateService.templatePrep["user-outbox"]
	require.True(t, exists, "user-outbox template not found")

	action, exists := template.Action("follower-unsubscribe")
	require.True(t, exists, "user-outbox must define a follower-unsubscribe action")
	require.Contains(t, action.Roles, model.MagicRoleAnonymous,
		"follower-unsubscribe must not be gated on a role: its audience never has one")

	// The recipient: an email address, no account, no session
	authorization := model.NewAuthorization()
	require.False(t, authorization.IsAuthenticated(), "the test subject must be a true anonymous visitor")

	// Somebody else's outbox -- an email follower is not the profile owner
	owner := model.NewUser()

	permissionService := NewPermission()
	allowed, err := permissionService.UserCan(nil, &authorization, &template, &owner, "follower-unsubscribe")

	require.Nil(t, err)
	require.True(t, allowed, "an email recipient must be able to reach their own unsubscribe link")
}
