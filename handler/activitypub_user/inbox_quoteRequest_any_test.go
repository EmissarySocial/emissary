package activitypub_user

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// quoteRequestFixture returns a QuoteRequest from Bob, and the quote post it carries, both shaped like Mastodon's
func quoteRequestFixture() (streams.Document, mapof.Any) {

	quote := mapof.Any{
		vocab.PropertyID:           "https://remote.example/users/bob/statuses/1",
		vocab.PropertyType:         "Note",
		vocab.PropertyAttributedTo: "https://remote.example/users/bob",
		vocab.PropertyQuote:        "https://local.example/my-post",
	}

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyID:         "https://remote.example/users/bob#quote_requests/1",
		vocab.PropertyType:       vocab.ActivityTypeQuoteRequest,
		vocab.PropertyActor:      "https://remote.example/users/bob",
		vocab.PropertyObject:     "https://local.example/my-post",
		vocab.PropertyInstrument: quote,
	})

	return activity, quote
}

// TestInbox_ValidateQuotePost pins every rule that a quote post must pass before its quote is approved
func TestInbox_ValidateQuotePost(t *testing.T) {

	// requireRefused asserts that one change to a valid quote post makes it invalid
	requireRefused := func(name string, change func(quote mapof.Any)) {
		t.Helper()

		activity, quote := quoteRequestFixture()
		change(quote)

		err := inbox_ValidateQuotePost(activity, streams.NewDocument(quote))
		require.Error(t, err, name)
		require.True(t, derp.IsValidationError(err), name)
	}

	// A quote post written by the sender, on the sender's server, quoting the requested object
	activity, quote := quoteRequestFixture()
	require.NoError(t, inbox_ValidateQuotePost(activity, streams.NewDocument(quote)))

	// The compatibility properties that FEP-044f reads are accepted too
	delete(quote, vocab.PropertyQuote)
	quote[vocab.PropertyQuoteURL] = "https://local.example/my-post"
	require.NoError(t, inbox_ValidateQuotePost(activity, streams.NewDocument(quote)), "quoteUrl")

	requireRefused("no id", func(quote mapof.Any) { delete(quote, vocab.PropertyID) })
	requireRefused("another server", func(quote mapof.Any) { quote[vocab.PropertyID] = "https://elsewhere.example/statuses/1" })
	requireRefused("a lookalike host", func(quote mapof.Any) { quote[vocab.PropertyID] = "https://remote.example.evil/statuses/1" })
	requireRefused("someone else's post", func(quote mapof.Any) { quote[vocab.PropertyAttributedTo] = "https://remote.example/users/carol" })
	requireRefused("no author", func(quote mapof.Any) { delete(quote, vocab.PropertyAttributedTo) })
	requireRefused("quotes another post", func(quote mapof.Any) { quote[vocab.PropertyQuote] = "https://local.example/other-post" })
	requireRefused("quotes nothing", func(quote mapof.Any) { delete(quote, vocab.PropertyQuote) })
}

// TestInbox_ValidateQuotePost_EmptyDocuments pins that empty documents are refused rather than matched,
// because an empty actor, object, and quote would otherwise compare equal
func TestInbox_ValidateQuotePost_EmptyDocuments(t *testing.T) {

	empty := streams.NewDocument(mapof.Any{})
	require.Error(t, inbox_ValidateQuotePost(empty, empty))

	// A quote post with an id but nothing else, from a request with no actor or object
	quote := streams.NewDocument(mapof.Any{vocab.PropertyID: "https://remote.example/statuses/1"})
	require.Error(t, inbox_ValidateQuotePost(empty, quote))
}
