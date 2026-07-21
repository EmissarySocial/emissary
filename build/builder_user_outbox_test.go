package build

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestOutboxBuilder(t *testing.T) {
	var _ StateSetter = Outbox{}

	// FIX #2: the admin User builder must also be a StateSetter, or the `set-state` step that
	// activates admin-created accounts (moving them to "LIVE") would fail at runtime.
	var _ StateSetter = User{}
}

// TestReportedBug_NonOwnerProfileNo500 is the regression test for the reported HTTP 500
// ("Unable to create Outbox builder") when viewing another user's profile.
//
// Root cause: a pre-LIVE profile rendered the onboarding wizard branch for EVERY viewer, and
// that branch calls the roles:["self"] "wizard-1" sub-view, which fails to build for non-owners.
// FIX #1 changed outbox.html's gate to `{{ if and (ne .StateID "LIVE") .IsMyself }}` so only the
// profile owner is routed into the wizard. This test exercises that exact conditional against real
// Outbox builders, proving non-owners and anonymous viewers never reach the self-only wizard.
func TestReportedBug_NonOwnerProfileNo500(t *testing.T) {

	// The literal gate from _embed/templates/user-outbox/outbox.html.
	gate := template.Must(template.New("gate").Parse(
		`{{- if and (ne .StateID "LIVE") .IsMyself -}}WIZARD{{- else -}}PROFILE{{- end -}}`))

	render := func(builder Builder) string {
		var buffer strings.Builder
		require.Nil(t, gate.Execute(&buffer, builder))
		return buffer.String()
	}

	// A public profile whose owner never completed onboarding (StateID is not "LIVE").
	ownerID := primitive.NewObjectID()
	pending := func(viewer primitive.ObjectID) Outbox {
		return Outbox{
			_user: &model.User{UserID: ownerID, IsPublic: true, StateID: ""},
			CommonWithTemplate: CommonWithTemplate{
				Common: Common{_authorization: model.Authorization{UserID: viewer}},
			},
		}
	}

	// The owner viewing their own pending profile still sees the onboarding wizard.
	require.Equal(t, "WIZARD", render(pending(ownerID)))

	// The reported bug: another logged-in user must fall through to the profile, NOT the wizard.
	require.Equal(t, "PROFILE", render(pending(primitive.NewObjectID())))

	// An anonymous visitor (zero UserID) must likewise see the profile.
	require.Equal(t, "PROFILE", render(pending(primitive.NilObjectID)))

	// Once activated (StateID == "LIVE"), even the owner sees the normal profile.
	live := Outbox{
		_user: &model.User{UserID: ownerID, IsPublic: true, StateID: "LIVE"},
		CommonWithTemplate: CommonWithTemplate{
			Common: Common{_authorization: model.Authorization{UserID: ownerID}},
		},
	}
	require.Equal(t, "PROFILE", render(live))
}

// TestOutbox_IsIndexable proves the profile's "Index on Search Engines" setting
// drives Outbox.IsIndexable, which overrides the indexable-by-default Common.
func TestOutbox_IsIndexable(t *testing.T) {

	// A brand-new/opted-out profile (isIndexable=false) is NOT indexable...
	optedOut := Outbox{_user: &model.User{IsIndexable: false}}
	require.False(t, optedOut.IsIndexable())

	// ...while a profile that opted in IS indexable.
	optedIn := Outbox{_user: &model.User{IsIndexable: true}}
	require.True(t, optedIn.IsIndexable())

	// Every other page type inherits the Common default: indexable.
	require.True(t, Common{}.IsIndexable())
}

// TestReportedBug_ProfileNoIndexDirective ties the fix back to the report: a
// public profile with isIndexable=false must emit a "noindex" robots directive
// in the page <head>, and an indexable profile must not. This exercises the
// exact conditional the theme includes-head.html templates render.
func TestReportedBug_ProfileNoIndexDirective(t *testing.T) {

	head := template.Must(template.New("head").Parse(
		`{{- if not .IsIndexable }}<meta name="robots" content="noindex">{{- end }}`))

	render := func(builder Builder) string {
		var buffer strings.Builder
		require.Nil(t, head.Execute(&buffer, builder))
		return buffer.String()
	}

	// The reported bug: opting out must now produce the directive.
	optedOut := render(Outbox{_user: &model.User{IsPublic: true, IsIndexable: false}})
	require.Equal(t, `<meta name="robots" content="noindex">`, optedOut)

	// Opting in emits nothing (default-indexable, no directive).
	optedIn := render(Outbox{_user: &model.User{IsPublic: true, IsIndexable: true}})
	require.Equal(t, "", optedIn)
}

// The following tests pin the viewer-permission gate on the User profile's outbox
// listings. Both Outbox() and Replies() render model.StreamSummary content
// (ContentHTML, Label, URL) to callers -- including anonymous ones, since the
// `outbox`/`replied` actions carry roles:["anonymous"]. The security-critical piece
// is Common.defaultAllowed(): without it, gated (circle/paid/non-anonymous) posts
// leak to anonymous visitors. Replies() historically omitted the filter (the "replied"
// tab leaked gated reply content) -- these tests lock both listings to the same gate.

// newAnonymousOutbox builds an Outbox builder for the given user as seen by an
// anonymous visitor (no authorization), with a minimal GET request. It relies on
// stubPermissionFactory (see step_ViewFeed_test.go) for the Permission and Stream
// services, so it needs no database.
func newAnonymousOutbox(userID primitive.ObjectID, target string) Outbox {

	request := httptest.NewRequest(http.MethodGet, target, nil)

	return Outbox{
		_user: &model.User{UserID: userID},
		CommonWithTemplate: CommonWithTemplate{
			Common: Common{
				_factory:       stubPermissionFactory{},
				_request:       request,
				_authorization: model.Authorization{}, // anonymous: no user, no groups
			},
		},
	}
}

// requireAnonymousGate asserts that the given criteria restrict results to
// anonymously-viewable, non-deleted streams -- the shared invariant for every
// public outbox listing.
func requireAnonymousGate(t *testing.T, criteria exp.Expression) {

	t.Helper()

	predicates := collectPredicates(criteria)

	// RULE: results MUST be filtered by the viewer's permission set, and an
	// anonymous viewer only carries the Anonymous magic group.
	predicate, exists := findPredicate(predicates, "defaultAllow", exp.OperatorIn)
	require.True(t, exists, "anonymous outbox listing MUST filter by defaultAllow")

	permissions, ok := predicate.Value.(model.Permissions)
	require.True(t, ok, "defaultAllow filter value must be a Permissions slice")
	require.Contains(t, permissions, model.MagicGroupIDAnonymous, "anonymous viewer must match anonymous-allowed streams")
	require.NotContains(t, permissions, model.MagicGroupIDAuthenticated, "anonymous viewer must NOT carry the authenticated group")

	// RULE: deleted streams are always excluded (defaultAllowed() carries this guard).
	_, hasDeleteGuard := findPredicate(predicates, "deleteDate", exp.OperatorEqual)
	require.True(t, hasDeleteGuard, "must exclude deleted streams")
}

// TestOutbox_Replies_FiltersByPermission is the regression test for the "replied" tab
// leak: Outbox.Replies() omitted w.defaultAllowed(), so anonymous callers to
// /@victim/replied saw the victim's gated (circle/paid) reply content.
func TestOutbox_Replies_FiltersByPermission(t *testing.T) {

	userID := primitive.NewObjectID()
	builder := newAnonymousOutbox(userID, "https://host/@victim/replied")

	criteria := builder.Replies().criteria
	predicates := collectPredicates(criteria)

	// RULE: gated replies must be permission-filtered exactly like the main outbox.
	requireAnonymousGate(t, criteria)

	// RULE: the reply listing is scoped to this user's outbox and to reply posts only.
	parentPredicate, hasParent := findPredicate(predicates, "parentId", exp.OperatorEqual)
	require.True(t, hasParent, "Replies() must scope to the user's outbox")
	require.Equal(t, userID, parentPredicate.Value, "parentId must equal the profile user's ID")

	inReplyTo, hasReply := findPredicate(predicates, "inReplyTo", exp.OperatorNotEqual)
	require.True(t, hasReply, "Replies() must list only reply posts")
	require.Equal(t, "", inReplyTo.Value, "inReplyTo filter must exclude non-replies")
}

// TestOutbox_Outbox_FiltersByPermission locks the sibling listing to the same gate,
// so a future refactor can't quietly drop defaultAllowed() from either method.
func TestOutbox_Outbox_FiltersByPermission(t *testing.T) {

	userID := primitive.NewObjectID()
	builder := newAnonymousOutbox(userID, "https://host/@victim/outbox")

	criteria := builder.Outbox().criteria
	predicates := collectPredicates(criteria)

	// RULE: the sibling listing shares the same anonymous gate.
	requireAnonymousGate(t, criteria)

	// RULE: the main outbox lists only top-level posts (inReplyTo == "").
	inReplyTo, hasReply := findPredicate(predicates, "inReplyTo", exp.OperatorEqual)
	require.True(t, hasReply, "Outbox() must list only top-level posts")
	require.Equal(t, "", inReplyTo.Value)
}
