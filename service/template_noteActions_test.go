package service

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// noteActionsBuilder is the smallest dot that a Note's action bar renders against
type noteActionsBuilder struct {
	signedIn      bool
	repliesClosed bool
}

// UserCan returns TRUE for the local Like button only when the simulated visitor is signed in here
func (builder noteActionsBuilder) UserCan(actionID string) bool {
	return builder.signedIn && (actionID == "like-button")
}

// IsAuthenticated returns TRUE when the simulated visitor is signed in here
func (builder noteActionsBuilder) IsAuthenticated() bool {
	return builder.signedIn
}

// AcceptsReplies returns FALSE when the simulated Note's replies are closed
func (builder noteActionsBuilder) AcceptsReplies() bool {
	return !builder.repliesClosed
}

// View returns a marker naming the embedded action, in place of rendering it
func (builder noteActionsBuilder) View(actionID string) (template.HTML, error) {
	return template.HTML("[view:" + actionID + "]"), nil // #nosec G203 -- a fixed test marker
}

// StreamID returns the simulated Note's ID
func (builder noteActionsBuilder) StreamID() string {
	return "0123456789abcdef01234567"
}

// Permalink returns the simulated Note's ActivityPub ID
func (builder noteActionsBuilder) Permalink() string {
	return "https://example.social/0123456789abcdef01234567"
}

// renderNoteActions executes the Note template's action bar against the stub builder
func renderNoteActions(t *testing.T, builder noteActionsBuilder) string {

	t.Helper()

	templateService := loadProfileTabsTemplates(t, nil)
	note, exists := templateService.templatePrep["outbox-message"]
	require.True(t, exists, "outbox-message is not loaded")

	var buffer bytes.Buffer
	require.NoError(t, note.HTMLTemplate.ExecuteTemplate(&buffer, "note-actions", builder))

	return buffer.String()
}

// TestNoteActions_OneClickTwins confirms that Like and Share each have a modal button and a hidden one-click twin
func TestNoteActions_OneClickTwins(t *testing.T) {

	html := renderNoteActions(t, noteActionsBuilder{})

	// Each intent is declared once, above its own pair, so CamperJS judges each button alone
	require.Contains(t, html, `data-camper-intent="like" data-camper-object="https://example.social/0123456789abcdef01234567"`)
	require.Contains(t, html, `data-camper-intent="announce" data-camper-object="https://example.social/0123456789abcdef01234567"`)

	// Both look the same, and only the modal's shows without CamperJS
	require.Equal(t, 2, strings.Count(html, "</span> Like</button>"))
	require.Equal(t, 2, strings.Count(html, "</span> Share</button>"))
	require.Equal(t, 2, strings.Count(html, `class="camper-hide-if-ready" hx-get="/0123456789abcdef01234567/intent?intent=`))
	require.Equal(t, 2, strings.Count(html, `class="camper-show-if-ready" data-camper-action="intent" hidden>`))

	// RULE: The "..." button opens the account manager, and stays hidden until an account is saved
	require.Contains(t, html, `class="camper-show-if-has-accounts" hx-get="/0123456789abcdef01234567/intent" aria-label="Manage Accounts" hidden>`)

	// The bar loads CamperJS itself
	require.Contains(t, html, `src="/.themes/global/resources/camperjs-0.2.0/camper.js"`)
}

// TestNoteActions_ShareIsAnnounce confirms that Share announces the Note itself, and never writes a new post about it
func TestNoteActions_ShareIsAnnounce(t *testing.T) {

	html := renderNoteActions(t, noteActionsBuilder{})

	require.Contains(t, html, `class="camper-hide-if-ready" hx-get="/0123456789abcdef01234567/intent?intent=announce&object=https://example.social/0123456789abcdef01234567"`)
	require.NotContains(t, html, `data-camper-intent="create"`)
	require.NotContains(t, html, "intent=create&content=")
}

// TestNoteActions_LocalLikeButton confirms that a visitor signed in here gets the local Like button, and no intents
func TestNoteActions_LocalLikeButton(t *testing.T) {

	html := renderNoteActions(t, noteActionsBuilder{signedIn: true})

	require.Contains(t, html, "[view:like-button]")
	require.Contains(t, html, `class="primary turboclick" hx-target="main" hx-swap="outerHTML" hx-get="/@me/newsfeed/browse-reply?url=https%3A%2F%2Fexample.social%2F0123456789abcdef01234567"`, "Reply opens the visitor's own inbox")
	require.NotContains(t, html, "camper-")
	require.NotContains(t, html, "camper.js")
}

// TestNoteActions_Reply confirms that Reply leads the bar, with a one-click twin, and only while the Note accepts replies
func TestNoteActions_Reply(t *testing.T) {

	html := renderNoteActions(t, noteActionsBuilder{})

	// The Reply intent declares both the post it answers and the object to fall back on
	require.Contains(t, html, `data-camper-intent="reply" data-camper-inreplyto="https://example.social/0123456789abcdef01234567" data-camper-object="https://example.social/0123456789abcdef01234567"`)
	// Reply is the bar's primary button, in both copies
	require.Contains(t, html, `class="primary turboclick camper-hide-if-ready" hx-get="/0123456789abcdef01234567/intent?intent=create&inReplyTo=`)
	require.Contains(t, html, `class="primary camper-show-if-ready" data-camper-action="intent" hidden><span aria-hidden="true" class="margin-left-xs"></span> Reply</button>`)
	require.Equal(t, 2, strings.Count(html, "</span> Reply</button>"))

	// Reply comes first, set apart from Like
	require.Less(t, strings.Index(html, "Reply</button>"), strings.Index(html, `data-camper-intent="like"`))
	require.Regexp(t, `Reply</button>\s*</span>\s+<span data-camper-intent="like"`, html)

	// RULE: A Note that refuses replies offers none
	closed := renderNoteActions(t, noteActionsBuilder{repliesClosed: true})
	require.NotContains(t, closed, "Reply</button>")
	require.Contains(t, closed, "</span> Like</button>")
}
