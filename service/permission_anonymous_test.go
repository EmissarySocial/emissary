package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Anonymous Authorization Tests
 *
 * roles:["anonymous"] grants an action to EVERY caller, and the
 * grant ignores the HTTP method: GET and POST both reach
 * Permission.UserCan, which takes no method.  A nil data.Session
 * is safe here, because the anonymous path returns first.
 ******************************************/

// anonymousTemplate builds a Template with the named action granted to the provided roles,
// then runs CalcAccessList the same way service.Template does at load time.
func anonymousTemplate(t *testing.T, actionID string, states []string, roles ...string) model.Template {

	t.Helper()

	// The harness in permission_authorization_test.go assigns AccessList directly; this one
	// goes through the calculation, because the calculation is part of what is under test.

	action := model.NewAction()
	action.Roles = roles
	action.States = states

	template := model.NewTemplate("test-template", nil)
	template.States = mapof.Object[model.State]{
		"default":   {StateID: "default"},
		"published": {StateID: "published"},
	}
	template.Actions = mapof.Object[model.Action]{
		actionID: action,
	}

	loaded := template.Actions[actionID]
	loaded.CalcAccessList(&template)
	template.Actions[actionID] = loaded

	return template
}

// publishedStream builds a Stream in the "published" state with no group, circle, or product grants
func publishedStream() model.Stream {
	stream := model.NewStream()
	stream.StateID = "published"
	return stream
}

// TestUserCan_Anonymous_GrantsUnauthenticatedVisitor verifies that roles:["anonymous"] admits a
// visitor with no session at all.
func TestUserCan_Anonymous_GrantsUnauthenticatedVisitor(t *testing.T) {

	// This is the grant a contact form's "submit" action relies on
	stream := publishedStream()
	template := anonymousTemplate(t, "submit", nil, model.MagicRoleAnonymous)

	permissionService := NewPermission()
	authorization := model.NewAuthorization()

	require.False(t, authorization.IsAuthenticated(), "the test subject must be a true anonymous visitor")

	allowed, err := permissionService.UserCan(nil, &authorization, &template, &stream, "submit")

	require.Nil(t, err)
	require.True(t, allowed, "roles:[\"anonymous\"] must admit an unauthenticated visitor")
}

// TestUserCan_Anonymous_GrantsEveryCaller verifies that "anonymous" means everyone, not
// "only signed-out users" -- a signed-in User must not be locked out of a public action.
func TestUserCan_Anonymous_GrantsEveryCaller(t *testing.T) {

	stream := publishedStream()
	template := anonymousTemplate(t, "submit", nil, model.MagicRoleAnonymous)
	permissionService := NewPermission()

	anonymous := model.NewAuthorization()

	signedIn := model.NewAuthorization()
	signedIn.UserID = primitive.NewObjectID()

	owner := model.NewAuthorization()
	owner.UserID = primitive.NewObjectID()
	owner.DomainOwner = true

	for name, authorization := range map[string]*model.Authorization{
		"anonymous":    &anonymous,
		"signed-in":    &signedIn,
		"domain owner": &owner,
	} {
		t.Run(name, func(t *testing.T) {
			allowed, err := permissionService.UserCan(nil, authorization, &template, &stream, "submit")
			require.Nil(t, err)
			require.True(t, allowed, "%s must be admitted by an anonymous action", name)
		})
	}
}

// TestCalcAccessList_AnonymousOverridesOtherRoles verifies that "anonymous" collapses the
// AccessList to itself, whatever other roles the action names.
func TestCalcAccessList_AnonymousOverridesOtherRoles(t *testing.T) {

	// So a Template meaning to restrict an action must not name "anonymous" at all
	template := anonymousTemplate(t, "submit", nil, model.MagicRoleAuthor, model.MagicRoleAnonymous)
	action := template.Actions["submit"]

	require.Equal(t, sliceof.String{model.MagicRoleAnonymous}, action.AccessList["published"])

	stream := publishedStream()
	permissionService := NewPermission()
	authorization := model.NewAuthorization()

	allowed, err := permissionService.UserCan(nil, &authorization, &template, &stream, "submit")
	require.Nil(t, err)
	require.True(t, allowed, "\"anonymous\" alongside \"author\" grants everyone")
}

// TestUserCan_Anonymous_StillHonorsStates verifies that an anonymous grant is scoped to the
// states the action declares.
func TestUserCan_Anonymous_StillHonorsStates(t *testing.T) {

	// A contact form gated to "published" must refuse a draft page
	template := anonymousTemplate(t, "submit", []string{"published"}, model.MagicRoleAnonymous)
	permissionService := NewPermission()
	authorization := model.NewAuthorization()

	// A published Stream is open to anonymous visitors
	published := publishedStream()
	allowed, err := permissionService.UserCan(nil, &authorization, &template, &published, "submit")
	require.Nil(t, err)
	require.True(t, allowed, "an anonymous action must run in a state it declares")

	// The SAME action, on a Stream in an undeclared state, is refused
	draft := model.NewStream()
	draft.StateID = "default"
	allowed, err = permissionService.UserCan(nil, &authorization, &template, &draft, "submit")
	require.Nil(t, err)
	require.False(t, allowed, "an anonymous action must NOT run in a state it does not declare")
}
