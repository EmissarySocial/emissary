package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// noteRepliesBuilder is the smallest dot that a Note's replies list renders against
type noteRepliesBuilder struct {
	links     []model.CollectionItem
	documents map[string]streams.Document
}

// ReplyLinksAfter returns the simulated Note's replies, all on one page
func (builder noteRepliesBuilder) ReplyLinksAfter(string, int) (sliceof.Object[model.CollectionItem], error) {
	return builder.links, nil
}

// ActivityStream returns a simulated document, or an empty one for a server that did not answer
func (builder noteRepliesBuilder) ActivityStream(url string) streams.Document {

	if document, exists := builder.documents[url]; exists {
		return document
	}

	return streams.NilDocument()
}

// QueryParam returns no query values, as for a Replies tab opened without ?reveal
func (builder noteRepliesBuilder) QueryParam(string) string {
	return ""
}

// StreamID returns the simulated Note's ID
func (builder noteRepliesBuilder) StreamID() string {
	return "0123456789abcdef01234567"
}

// IsAuthenticated returns FALSE, for a visitor who is not signed in here
func (builder noteRepliesBuilder) IsAuthenticated() bool {
	return false
}

// renderNoteReplies executes the Note template's replies list against the stub builder
func renderNoteReplies(t *testing.T, builder noteRepliesBuilder) string {

	t.Helper()

	templateService := loadProfileTabsTemplates(t, nil)
	note, exists := templateService.templatePrep["outbox-message"]
	require.True(t, exists, "outbox-message is not loaded")

	var buffer bytes.Buffer
	require.NoError(t, note.HTMLTemplate.ExecuteTemplate(&buffer, "replies", builder))

	return buffer.String()
}

// replyDocument returns a simulated reply by the named author
func replyDocument(id string, author string, content string) streams.Document {
	return streams.NewDocument(map[string]any{"id": id, "type": "Note", "attributedTo": author, "content": content})
}

// TestNoteReplies_DrawsEveryReplyInOneRequest confirms that the list arrives whole, with no placeholder that loads later
func TestNoteReplies_DrawsEveryReplyInOneRequest(t *testing.T) {

	builder := noteRepliesBuilder{
		links: []model.CollectionItem{{URI: "https://example.com/replies/1"}, {URI: "https://example.com/replies/2"}},
		documents: map[string]streams.Document{
			"https://example.com/replies/1": replyDocument("https://example.com/replies/1", "https://example.com/users/amy", "First reply"),
			"https://example.com/replies/2": replyDocument("https://example.com/replies/2", "https://example.com/users/bo", "Second reply"),
			"https://example.com/users/amy": streams.NewDocument(map[string]any{"id": "https://example.com/users/amy", "type": "Person", "name": "Amy"}),
			"https://example.com/users/bo":  streams.NewDocument(map[string]any{"id": "https://example.com/users/bo", "type": "Person", "name": "Bo"}),
		},
	}

	html := renderNoteReplies(t, builder)

	require.Contains(t, html, "First reply")
	require.Contains(t, html, "Second reply")
	require.Contains(t, html, `<span class="bold margin-right-xs">Amy</span>`, "the author is loaded through .ActivityStream")
	require.Contains(t, html, `<span class="bold margin-right-xs">Bo</span>`)
	require.Less(t, strings.Index(html, "First reply"), strings.Index(html, "Second reply"), "replies keep their order")

	// RULE: Nothing in the list loads later, so its height never changes after it arrives
	require.NotContains(t, html, `hx-trigger="load"`)
	require.NotContains(t, html, "height:100px")

	// goToUrl is defined once for the whole list
	require.Equal(t, 1, strings.Count(html, "function goToUrl"))
}

// TestNoteReplies_AuthorWhoseServerIsGone confirms that one author who cannot be loaded leaves the rest of the list drawn
func TestNoteReplies_AuthorWhoseServerIsGone(t *testing.T) {

	builder := noteRepliesBuilder{
		links: []model.CollectionItem{{URI: "https://example.com/replies/1"}},
		documents: map[string]streams.Document{
			"https://example.com/replies/1": replyDocument("https://example.com/replies/1", "https://example.com/users/gone", "Still here"),
		},
	}

	html := renderNoteReplies(t, builder)

	require.Contains(t, html, "Still here")
	require.Contains(t, html, `<div class="circle width-48"></div>`, "an author with no icon gets the empty circle")
}

// TestNoteReplies_HiddenReply confirms that a reply the viewer's rules hide draws its placeholder, which reveals it in place
func TestNoteReplies_HiddenReply(t *testing.T) {

	hidden := streams.NewDocument(
		map[string]any{"id": "https://example.com/replies/1", "type": "Note", "content": "Secret"},
		streams.WithLabels(metadata.LabelSet{{Value: "Muted by your rules", IsHidden: true}}),
	)

	builder := noteRepliesBuilder{
		links:     []model.CollectionItem{{URI: "https://example.com/replies/1"}},
		documents: map[string]streams.Document{"https://example.com/replies/1": hidden},
	}

	html := renderNoteReplies(t, builder)

	require.NotContains(t, html, "Secret")
	require.Contains(t, html, `hx-get="/0123456789abcdef01234567/reply?id=https%3A%2F%2Fexample.com%2Freplies%2F1&reveal=true"`)
	require.Contains(t, html, "Select to show")
}
