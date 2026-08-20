package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests lock in the authorization decision that the Mastodon-compat status
// handlers rely on (GetStatus, GetStatus_Source, PostStatus_Translate). Each of
// those handlers loads a Stream by URL and then calls Permission.UserCan(..., "view")
// before returning any content. The regression we are guarding against is an IDOR:
// a token belonging to user A must NOT be able to read a Stream authored by user B
// when the Stream's "view" action is gated to its author.
//
// UserCan is exercised directly (rather than through the handler) because the mastodon
// handlers build a full domain Factory with no injection seam. The author/anonymous/
// group paths through UserCan never touch the data.Session, so a nil session is safe
// here; the identity-privilege path (IsIdentity) is the only branch that reads it, and
// a plain user Authorization never reaches it.

// authorStream builds a Stream authored by authorID whose "view" action is granted
// only to the "author" role in the stream's current state.
func authorStream(authorID primitive.ObjectID) model.Stream {
	stream := model.NewStream()
	stream.StateID = "default"
	stream.AttributedTo = model.PersonLink{UserID: authorID}
	return stream
}

// viewTemplate builds a Template whose "view" action grants the provided roles
// in the "default" state.
func viewTemplate(roles ...string) model.Template {
	return actionTemplate("view", roles...)
}

// actionTemplate builds a Template whose named action grants the provided roles
// in the "default" state. AccessList is normally computed by Action.CalcAccessList
// from Roles/States/StateRoles; here we set it directly, which is exactly what
// UserCan consumes.
func actionTemplate(actionID string, roles ...string) model.Template {
	action := model.NewAction()
	action.AccessList = mapof.Object[sliceof.String]{
		"default": sliceof.String(roles),
	}

	template := model.NewTemplate("test-template", nil)
	template.Actions = mapof.Object[model.Action]{
		actionID: action,
	}

	return template
}

// TestUserCan_View_AuthorOnly_DeniesOtherUser verifies that an author-gated Stream refuses a different signed-in User
func TestUserCan_View_AuthorOnly_DeniesOtherUser(t *testing.T) {

	author := primitive.NewObjectID()
	otherUser := primitive.NewObjectID()

	stream := authorStream(author)
	template := viewTemplate(model.MagicRoleAuthor)

	permissionService := NewPermission()

	// The author themselves may view the stream.
	authorAuth := model.NewAuthorization()
	authorAuth.UserID = author

	allowed, err := permissionService.UserCan(nil, &authorAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.True(t, allowed, "the author must be allowed to view their own stream")

	// A DIFFERENT authenticated user must be denied. This is the IDOR guard.
	otherAuth := model.NewAuthorization()
	otherAuth.UserID = otherUser

	allowed, err = permissionService.UserCan(nil, &otherAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.False(t, allowed, "a non-author user must NOT be allowed to view an author-gated stream")
}

// TestUserCan_AuthorGatedActions verifies that UserCan honors an author-only access
// list for any action, allowing the author and denying every other user. The Mastodon
// READ handlers (GetStatus, GetStatus_Source, PostStatus_Translate) rely on this for
// the "view" action; the "edit"/"delete" cases here document the same property for
// completeness. (The Mastodon WRITE handlers do NOT use UserCan — they are author-only
// via userOwnsStream; see handler/mastodon/status_authorization_test.go.)
func TestUserCan_AuthorGatedActions(t *testing.T) {

	permissionService := NewPermission()

	author := primitive.NewObjectID()
	otherUser := primitive.NewObjectID()

	authorAuth := model.NewAuthorization()
	authorAuth.UserID = author

	otherAuth := model.NewAuthorization()
	otherAuth.UserID = otherUser

	for _, actionID := range []string{"view", "edit", "delete"} {

		stream := authorStream(author)
		template := actionTemplate(actionID, model.MagicRoleAuthor)

		allowed, err := permissionService.UserCan(nil, &authorAuth, &template, &stream, actionID)
		require.Nil(t, err)
		require.True(t, allowed, "the author must be allowed to "+actionID+" their own stream")

		allowed, err = permissionService.UserCan(nil, &otherAuth, &template, &stream, actionID)
		require.Nil(t, err)
		require.False(t, allowed, "a non-author user must NOT be allowed to "+actionID+" an author-gated stream")
	}
}

// TestUserCan_View_AuthorOnly_DeniesAnonymous verifies that an author-gated Stream refuses an anonymous caller
func TestUserCan_View_AuthorOnly_DeniesAnonymous(t *testing.T) {

	author := primitive.NewObjectID()

	stream := authorStream(author)
	template := viewTemplate(model.MagicRoleAuthor)

	permissionService := NewPermission()

	// An anonymous (unauthenticated) caller must be denied.
	anonAuth := model.NewAuthorization()

	allowed, err := permissionService.UserCan(nil, &anonAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.False(t, allowed, "an anonymous caller must NOT be allowed to view an author-gated stream")
}

// TestUserCan_View_Anonymous_AllowsEveryone verifies that an anonymous-gated Stream is readable by anyone
func TestUserCan_View_Anonymous_AllowsEveryone(t *testing.T) {

	author := primitive.NewObjectID()
	otherUser := primitive.NewObjectID()

	stream := authorStream(author)
	template := viewTemplate(model.MagicRoleAnonymous)

	permissionService := NewPermission()

	// A public stream is viewable by an unrelated user...
	otherAuth := model.NewAuthorization()
	otherAuth.UserID = otherUser

	allowed, err := permissionService.UserCan(nil, &otherAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.True(t, allowed, "any user may view an anonymous-gated stream")

	// ...and by an anonymous caller.
	anonAuth := model.NewAuthorization()

	allowed, err = permissionService.UserCan(nil, &anonAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.True(t, allowed, "an anonymous caller may view an anonymous-gated stream")
}

// TestUserCan_View_DomainOwner_AlwaysAllowed verifies that a domain owner may view any Stream
func TestUserCan_View_DomainOwner_AlwaysAllowed(t *testing.T) {

	author := primitive.NewObjectID()

	stream := authorStream(author)
	template := viewTemplate(model.MagicRoleAuthor)

	permissionService := NewPermission()

	// A domain owner bypasses the author gate.
	ownerAuth := model.NewAuthorization()
	ownerAuth.UserID = primitive.NewObjectID()
	ownerAuth.DomainOwner = true

	allowed, err := permissionService.UserCan(nil, &ownerAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.True(t, allowed, "a domain owner may view any stream")
}

// TestUserCan_View_MissingAction_Denies verifies that a Template with no "view" action denies everyone
func TestUserCan_View_MissingAction_Denies(t *testing.T) {

	author := primitive.NewObjectID()

	stream := authorStream(author)

	// A template with no "view" action must deny (UserCan returns false when the
	// action is absent) rather than fall through to returning content.
	template := model.NewTemplate("test-template", nil)

	permissionService := NewPermission()

	authorAuth := model.NewAuthorization()
	authorAuth.UserID = author

	allowed, err := permissionService.UserCan(nil, &authorAuth, &template, &stream, "view")
	require.Nil(t, err)
	require.False(t, allowed, "a missing view action must deny access")
}

// TestStream_IsMyself_AlwaysFalse documents a trap: Stream.IsMyself is part of the
// AccessLister interface and, for a Stream, ALWAYS returns false (a Stream never
// directly represents a User the way a User profile object does). It must never be
// used as an "is this the author?" check in a handler — doing so would reject the
// real author too. Author checks go through UserCan with the relevant action (whose
// AccessList grants the "author" role), or model.Stream.IsAuthor.
func TestStream_IsMyself_AlwaysFalse(t *testing.T) {

	author := primitive.NewObjectID()
	stream := authorStream(author)

	require.False(t, stream.IsMyself(author), "Stream.IsMyself must return false even for the author")
	require.False(t, stream.IsMyself(primitive.NewObjectID()), "Stream.IsMyself must return false for any user")

	// IsAuthor is the correct author predicate.
	require.True(t, stream.IsAuthor(author), "IsAuthor must return true for the author")
	require.False(t, stream.IsAuthor(primitive.NewObjectID()), "IsAuthor must return false for a non-author")
}
