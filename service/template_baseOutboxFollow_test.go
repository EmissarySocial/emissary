package service

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/stretchr/testify/require"
)

// followActions are the base-outbox actions behind the Follow modal, including the three email pages Go links to
var followActions = []string{
	"follow", "follow-activitypub",
	"follow-email", "follow-email-sent",
	"follower-confirm", "follower-unsubscribe",
}

// followBuilder is the smallest dot that the base-outbox Follow views render against
type followBuilder struct{}

// UserID returns the simulated profile's ID
func (builder followBuilder) UserID() string {
	return "0123456789abcdef01234567"
}

// DisplayName returns the simulated profile's name
func (builder followBuilder) DisplayName() string {
	return "Alice"
}

// IconURL returns the simulated profile's avatar
func (builder followBuilder) IconURL() string {
	return "https://example.social/alice.webp"
}

// ProfileURL returns the simulated profile's page
func (builder followBuilder) ProfileURL() string {
	return "https://example.social/@alice"
}

// Username returns the simulated profile's username
func (builder followBuilder) Username() string {
	return "alice"
}

// Hostname returns the simulated server's hostname
func (builder followBuilder) Hostname() string {
	return "example.social"
}

// Permalink returns the simulated profile's ActivityPub ID
func (builder followBuilder) Permalink() string {
	return "https://example.social/@0123456789abcdef01234567"
}

// IsBridgeBluesky returns FALSE, so that the Bluesky button is left out
func (builder followBuilder) IsBridgeBluesky() bool {
	return false
}

// BlueskyHandle returns nothing, because the simulated profile is not bridged
func (builder followBuilder) BlueskyHandle() string {
	return ""
}

// TestBaseOutboxFollow_EveryOutboxGetsTheEmailPages confirms that an outbox gets the Follow actions from base-outbox, and a Stream does not
func TestBaseOutboxFollow_EveryOutboxGetsTheEmailPages(t *testing.T) {

	// An outbox that extends base-outbox and defines nothing to do with following, as Qwertylicious's does
	custom := fstest.MapFS{
		"template.hjson": &fstest.MapFile{Data: []byte(`{
			templateId: test-outbox
			templateRole: user-outbox
			category: Special
			model: Outbox
			extends: ["base-intent", "base-outbox"]
			states: {
				default: {
					label: "Default"
				}
			}
			actions: {}
		}`)},
	}

	templateService := loadProfileTabsTemplates(t, map[string]fstest.MapFS{"test-outbox": custom})

	// RULE: Go links to the email pages, so every outbox must answer them
	for _, templateID := range []string{"base-outbox", "user-outbox", "test-outbox"} {
		for _, actionID := range followActions {
			_, exists := templateService.templatePrep[templateID].Actions[actionID]
			require.True(t, exists, "%s is missing %q", templateID, actionID)
		}
	}

	// The modal is still a modal once inherited
	_, isModal := templateService.templatePrep["test-outbox"].Actions["follow"].Steps[0].(step.AsModal)
	require.True(t, isModal, "follow must open as a modal")

	// RULE: Streams extend base-intent, and cannot render these views, so base-intent holds none of them
	for _, actionID := range followActions {
		_, exists := templateService.templatePrep["base-intent"].Actions[actionID]
		require.False(t, exists, "base-intent must not define %q", actionID)
	}

	// So no outbox is reported missing any of them
	require.Empty(t, templateService.missingProfileTabs(), "every embedded outbox has the base-outbox actions")
}

// TestBaseOutboxFollow_OwnViewReplacesInherited confirms that an outbox restyles the Follow modal by naming a view
func TestBaseOutboxFollow_OwnViewReplacesInherited(t *testing.T) {

	// An outbox with a follow-heading of its own, as Atlas's outbox has
	custom := fstest.MapFS{
		"template.hjson": &fstest.MapFile{Data: []byte(`{
			templateId: test-outbox
			templateRole: user-outbox
			category: Special
			model: Outbox
			extends: ["base-outbox"]
			states: {
				default: {
					label: "Default"
				}
			}
			actions: {}
		}`)},
		"follow-heading.html": &fstest.MapFile{Data: []byte(`<h1 id="custom-heading">{{.DisplayName}}</h1>`)},
	}

	templateService := loadProfileTabsTemplates(t, map[string]fstest.MapFS{"test-outbox": custom})

	// Its heading heads the inherited email form
	html := renderFollowView(t, templateService, "test-outbox", "follow-email")
	require.Contains(t, html, `<h1 id="custom-heading">Alice</h1>`)
	require.Contains(t, html, `action="/.follower/new"`, "the inherited form must still render")

	// The stock heading is untouched for everyone else
	require.NotContains(t, renderFollowView(t, templateService, "user-outbox", "follow-email"), "custom-heading")
}

// TestBaseOutboxFollow_AccountListsCarryTheirStyles confirms that every Follow view listing accounts styles them itself
func TestBaseOutboxFollow_AccountListsCarryTheirStyles(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)

	// Each view replaces the modal, so the styles of the view before it are gone
	for _, view := range []string{"follow", "follow-activitypub"} {

		html := renderFollowView(t, templateService, "user-outbox", view)

		require.Contains(t, html, "camper-accounts", "%s lists accounts", view)
		require.Contains(t, html, ".camper-account-icon", "%s must include follow-styles", view)
		require.Contains(t, html, `data-camper-object="https://example.social/@0123456789abcdef01234567"`, view)
	}

	// RULE: A view added later that lists accounts must include the styles too
	files, err := filepath.Glob("../_embed/templates/base-outbox/*.html")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {

		content, err := os.ReadFile(file) // #nosec G304 -- the names come from a glob of one embedded folder
		require.NoError(t, err)

		if strings.Contains(string(content), "camper-accounts") {
			require.Contains(t, string(content), `{{- template "follow-styles" -}}`, file)
		}
	}
}

// TestBaseOutboxFollow_FollowButton confirms that the Follow button draws both copies, and only the modal's shows without CamperJS
func TestBaseOutboxFollow_FollowButton(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)
	html := renderFollowView(t, templateService, "user-outbox", "follow-button")

	// Both copies look the same
	require.Equal(t, 2, strings.Count(html, "⁂ Follow Alice"))

	// The intent and object sit above both copies, so CamperJS reads them from either
	require.Contains(t, html, `data-camper-intent="follow" data-camper-object="https://example.social/@0123456789abcdef01234567"`)

	// RULE: Without CamperJS, only the copy that opens the Follow modal shows
	require.Contains(t, html, `camper-hide-if-ready" hx-get="/@0123456789abcdef01234567/follow"`)
	require.Regexp(t, `<div class="camper-show-if-ready[^"]*" hidden>`, html)

	// The one-click copy follows, and a "..." button beside it opens the account manager
	require.Contains(t, html, `data-camper-action="intent"`)
	require.Contains(t, html, `hx-get="/@0123456789abcdef01234567/intent"`)
	require.Contains(t, html, `src="/.themes/global/resources/camperjs-0.2.0/camper.js"`)
}

// TestBaseOutboxFollow_OneAccount confirms that the Follow modal offers to add a Fediverse account only while none is saved
func TestBaseOutboxFollow_OneAccount(t *testing.T) {

	templateService := loadProfileTabsTemplates(t, nil)

	// RULE: A visitor keeps one account, so the only way into follow-activitypub sits in the no-account section
	menu := renderFollowView(t, templateService, "user-outbox", "follow")
	require.Equal(t, 1, strings.Count(menu, `/follow-activitypub"`))
	require.Less(t, strings.Index(menu, `class="camper-hide-if-has-accounts"`), strings.Index(menu, `/follow-activitypub"`))

	// The form there hides once an account is saved, and declares no limit
	activityPub := renderFollowView(t, templateService, "user-outbox", "follow-activitypub")
	require.Contains(t, activityPub, `class="camper-form camper-add-account"`)
	require.NotContains(t, activityPub, "max-accounts")
}

// renderFollowView executes one Follow view of the named Template against the stub builder
func renderFollowView(t *testing.T, templateService *Template, templateID string, view string) string {

	t.Helper()

	template, exists := templateService.templatePrep[templateID]
	require.True(t, exists, "template %q is not loaded", templateID)

	var buffer bytes.Buffer
	require.NoError(t, template.HTMLTemplate.ExecuteTemplate(&buffer, view, followBuilder{}))

	return buffer.String()
}
