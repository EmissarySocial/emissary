package build

import (
	"html/template"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
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
