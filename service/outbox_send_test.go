package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// newQuoteRequest returns a QuoteRequest shaped like Mastodon's, with the quote post inlined
func newQuoteRequest() streams.Document {
	return streams.NewDocument(mapof.Any{
		vocab.PropertyID:     "https://remote.example/users/bob#quote_requests/1",
		vocab.PropertyType:   vocab.ActivityTypeQuoteRequest,
		vocab.PropertyActor:  "https://remote.example/users/bob",
		vocab.PropertyObject: "https://local.example/my-post",
		vocab.PropertyInstrument: mapof.Any{
			vocab.PropertyID:           "https://remote.example/users/bob/statuses/1",
			vocab.PropertyType:         "Note",
			vocab.PropertyAttributedTo: "https://remote.example/users/bob",
			vocab.PropertyContent:      "<p>Look at this</p>",
			vocab.PropertyQuote:        "https://local.example/my-post",
		},
	})
}

// TestAcceptQuoteActivity pins the Accept that answers a QuoteRequest: Mastodon matches it to the request
// by the embedded id, and reads the QuoteAuthorization from its result
func TestAcceptQuoteActivity(t *testing.T) {

	authorizationURL := "https://local.example/my-post/pub/quoteAuthorization/123412341234123412341234"
	accept := acceptQuoteActivity("https://local.example/@author", newQuoteRequest(), authorizationURL)

	require.Equal(t, sliceof.Any{vocab.ContextTypeActivityStreams, vocab.ContextTypeQuotePosts}, accept[vocab.AtContext])
	require.Equal(t, authorizationURL+"#accept", accept.GetString(vocab.PropertyID))
	require.Equal(t, vocab.ActivityTypeAccept, accept.GetString(vocab.PropertyType))
	require.Equal(t, "https://local.example/@author", accept.GetString(vocab.PropertyActor))
	require.Equal(t, authorizationURL, accept.GetString(vocab.PropertyResult))
	require.Equal(t, []string{"https://remote.example/users/bob"}, accept[vocab.PropertyTo])

	// The QuoteRequest is embedded by reference, and the inlined quote post is reduced to its id
	require.Equal(t, mapof.Any{
		vocab.PropertyID:         "https://remote.example/users/bob#quote_requests/1",
		vocab.PropertyType:       vocab.ActivityTypeQuoteRequest,
		vocab.PropertyActor:      "https://remote.example/users/bob",
		vocab.PropertyObject:     "https://local.example/my-post",
		vocab.PropertyInstrument: "https://remote.example/users/bob/statuses/1",
	}, accept[vocab.PropertyObject])
}

// TestAcceptQuoteActivity_LinkedInstrument pins that a QuoteRequest whose quote post is a bare link
// produces the same Accept as one that inlines it
func TestAcceptQuoteActivity_LinkedInstrument(t *testing.T) {

	quoteRequest := newQuoteRequest()
	quoteRequest.SetProperty(vocab.PropertyInstrument, "https://remote.example/users/bob/statuses/1")

	accept := acceptQuoteActivity("https://local.example/@author", quoteRequest, "https://local.example/my-post/pub/quoteAuthorization/1")
	object, ok := accept[vocab.PropertyObject].(mapof.Any)
	require.True(t, ok)
	require.Equal(t, "https://remote.example/users/bob/statuses/1", object.GetString(vocab.PropertyInstrument))
}

// TestDeleteQuoteAuthorizationActivity pins the Delete that revokes a quote, in the shape that routes
// to Mastodon's revoke_quote: an embedded object typed QuoteAuthorization, signed by the quoted author
func TestDeleteQuoteAuthorizationActivity(t *testing.T) {

	authorization := mapof.Any{
		vocab.AtContext:                 sliceof.Any{vocab.ContextTypeActivityStreams},
		vocab.PropertyID:                "https://local.example/my-post/pub/quoteAuthorization/1",
		vocab.PropertyType:              vocab.ObjectTypeQuoteAuthorization,
		vocab.PropertyAttributedTo:      "https://local.example/@author",
		vocab.PropertyInteractingObject: "https://remote.example/users/bob/statuses/1",
		vocab.PropertyInteractionTarget: "https://local.example/my-post",
	}

	remove := deleteQuoteAuthorizationActivity("https://local.example/@author", authorization, "https://remote.example/users/bob", "https://local.example/@author/pub/followers")

	require.Equal(t, "https://local.example/my-post/pub/quoteAuthorization/1#delete", remove.GetString(vocab.PropertyID))
	require.Equal(t, vocab.ActivityTypeDelete, remove.GetString(vocab.PropertyType))
	require.Equal(t, "https://local.example/@author", remove.GetString(vocab.PropertyActor))
	require.Equal(t, []string{vocab.NamespacePublic, "https://remote.example/users/bob"}, remove[vocab.PropertyTo])
	require.Equal(t, []string{"https://local.example/@author/pub/followers"}, remove[vocab.PropertyCC])

	// The embedded QuoteAuthorization keeps every field but its @context
	object, ok := remove[vocab.PropertyObject].(mapof.Any)
	require.True(t, ok)
	require.NotContains(t, object, vocab.AtContext)
	require.Equal(t, vocab.ObjectTypeQuoteAuthorization, object.GetString(vocab.PropertyType))
	require.Equal(t, "https://local.example/my-post/pub/quoteAuthorization/1", object.GetString(vocab.PropertyID))
	require.Len(t, object, 5)

	// The caller's QuoteAuthorization is not modified
	require.Contains(t, authorization, vocab.AtContext)
}

// TestDeleteQuoteAuthorizationActivity_NoQuoter pins that a Delete for a quote post that no longer
// names its author is addressed to Public and the followers, with no empty recipient
func TestDeleteQuoteAuthorizationActivity_NoQuoter(t *testing.T) {

	authorization := mapof.Any{vocab.PropertyID: "https://local.example/my-post/pub/quoteAuthorization/1"}
	remove := deleteQuoteAuthorizationActivity("https://local.example/@author", authorization, "", "https://local.example/@author/pub/followers")

	require.Equal(t, []string{vocab.NamespacePublic}, remove[vocab.PropertyTo])
	require.Equal(t, []string{"https://local.example/@author/pub/followers"}, remove[vocab.PropertyCC])
}

// TestOutbox_DeliverDeleteQuoteAuthorization pins that the Delete is rebuilt from a queued task's
// plain-string arguments into the same QuoteAuthorization that was served
func TestOutbox_DeliverDeleteQuoteAuthorization(t *testing.T) {

	tasks := postcommit.NewTasks()
	session := spooledSession{Session: itemSession{store: &itemStore{}}, tasks: tasks}
	service := Outbox{host: "https://local.example"}

	authorization := model.QuoteAuthorizationJSONLD(
		"https://local.example/my-post/pub/quoteAuthorization/1",
		"https://local.example/@author",
		"https://remote.example/users/bob/statuses/1",
		"https://local.example/my-post",
	)

	// Queue the lookup task, then deliver from the arguments it stored
	service.SendDeleteQuoteAuthorization(session, "https://local.example/@author", authorization, "https://local.example/@author/pub/followers")

	queued := tasks.Drain()
	require.Len(t, queued, 1)
	require.Equal(t, TaskSendDeleteQuoteAuthorization, queued[0].Name)

	for name, value := range queued[0].Arguments {
		require.IsType(t, "", value, "argument %q must be a string, which the stored queue returns unchanged", name)
	}

	service.DeliverDeleteQuoteAuthorization(session, queued[0].Arguments, "https://remote.example/users/bob")

	delivered := tasks.Drain()
	require.Len(t, delivered, 1)
	require.Equal(t, sender.OutboxSendToAllRecipients, delivered[0].Name)

	remove := delivered[0].Arguments
	require.Equal(t, "https://local.example/@author", remove.GetString(vocab.PropertyActor))
	require.Equal(t, []string{vocab.NamespacePublic, "https://remote.example/users/bob"}, remove[vocab.PropertyTo])

	object, ok := remove[vocab.PropertyObject].(mapof.Any)
	require.True(t, ok)
	delete(authorization, vocab.AtContext)
	require.Equal(t, authorization, object)
}
