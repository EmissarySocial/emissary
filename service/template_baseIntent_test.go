package service

import (
	"bytes"
	"html/template"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// intentModalBuilder stands in for the Builder that renders base-intent's modal, which reads only these two values
type intentModalBuilder struct {
	query url.Values
}

// QueryParam returns one query parameter of the simulated request
func (builder intentModalBuilder) QueryParam(name string) string {
	return builder.query.Get(name)
}

// BasePath returns the simulated path of the page that opened the modal
func (builder intentModalBuilder) BasePath() string {
	return "/@alice"
}

// renderIntentModal minifies, parses, and executes base-intent/intent.html the way the server does, for one query string
func renderIntentModal(t *testing.T, query string) string {

	t.Helper()

	content, err := os.ReadFile("../_embed/templates/base-intent/intent.html")
	require.NoError(t, err)

	m := minify.New()
	minifier := html.Minifier{KeepEndTags: true, KeepQuotes: true, KeepDocumentTags: true}
	m.AddFunc("text/html", minifier.Minify)

	minified, err := m.String("text/html", string(content))
	require.NoError(t, err)

	tmpl, err := template.New("intent").Funcs(testFuncMap()).Parse(minified)
	require.NoError(t, err)

	values, err := url.ParseQuery(query)
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, tmpl.Execute(&buffer, intentModalBuilder{query: values}))
	return buffer.String()
}

// TestBaseIntentModal_Like verifies that the intent name is lowercased and the object reaches CamperJS
func TestBaseIntentModal_Like(t *testing.T) {

	result := renderIntentModal(t, "intent=Like&object=https%3A%2F%2Fexample.social%2Fnote")

	require.Contains(t, result, `data-camper-intent="like"`)
	require.Contains(t, result, `data-camper-object="https://example.social/note"`)
	require.Contains(t, result, `data-camper-proxy-url="/.proxy"`)
	require.Contains(t, result, `hx-on:camper:hide=`)
	require.NotContains(t, result, "/follow-email")
}

// TestBaseIntentModal_CreateReply verifies that a Create answering a post becomes a Reply to that post
func TestBaseIntentModal_CreateReply(t *testing.T) {

	result := renderIntentModal(t, "intent=create&inReplyTo=https%3A%2F%2Fexample.social%2Fnote")

	require.Contains(t, result, `data-camper-intent="reply"`)
	require.Contains(t, result, `data-camper-object="https://example.social/note"`)
	require.Contains(t, result, `data-camper-inreplyto="https://example.social/note"`)
}

// TestBaseIntentModal_CreateShare verifies that a plain Create stays a Create and carries its content
func TestBaseIntentModal_CreateShare(t *testing.T) {

	result := renderIntentModal(t, "intent=create&content=Hello+world")

	require.Contains(t, result, `data-camper-intent="create"`)
	require.Contains(t, result, `data-camper-content="Hello world"`)
}

// TestBaseIntentModal_Follow verifies that only the Follow intent offers email updates
func TestBaseIntentModal_Follow(t *testing.T) {

	result := renderIntentModal(t, "intent=follow&object=https%3A%2F%2Fexample.social%2F%40alice")

	require.Contains(t, result, `data-camper-intent="follow"`)
	require.Contains(t, result, `hx-get="/@alice/follow-email"`)
}

// TestBaseIntentModal_Manage verifies that a modal opened with no intent lists accounts, and runs nothing when one is clicked
func TestBaseIntentModal_Manage(t *testing.T) {

	result := renderIntentModal(t, "")

	require.Contains(t, result, "Your Fediverse Accounts")
	require.Contains(t, result, `hx-on:camper:intent="event.preventDefault()"`)
	require.Contains(t, result, `data-camper-proxy-url="/.proxy"`)
	require.NotContains(t, result, "data-camper-intent=")
}

// TestBaseIntentModal_Invalid verifies that an unknown intent draws no account widgets
func TestBaseIntentModal_Invalid(t *testing.T) {

	for _, query := range []string{"intent=bogus", "intent=%22%3E%3Cscript%3E"} {
		result := renderIntentModal(t, query)

		require.Contains(t, result, "Invalid Intent", query)
		require.NotContains(t, result, "camper-accounts", query)
		require.NotContains(t, result, "<script>", query)
	}
}

// TestBaseIntentModal_EscapesParameters verifies that a hostile query value cannot leave its attribute
func TestBaseIntentModal_EscapesParameters(t *testing.T) {

	result := renderIntentModal(t, "intent=like&object=%22%3E%3Cimg+src%3Dx+onerror%3Dalert(1)%3E")

	require.NotContains(t, result, "<img")
	require.Contains(t, result, `data-camper-object="&#34;&gt;&lt;img src=x onerror=alert(1)&gt;"`)
}
