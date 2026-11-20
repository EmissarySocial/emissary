package activitypub_user

import (
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for inbound QuoteRequest activities (FEP-044f)
func init() {
	inboxRouter.Add(vocab.ActivityTypeQuoteRequest, vocab.Any, inbox_QuoteRequest)
}

// inbox_QuoteRequest approves a quote of one of this User's Streams whenever the Stream accepts quotes,
// and answers with an Accept whose result is the new QuoteAuthorization.  A refusal is silent.
func inbox_QuoteRequest(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.inbox_QuoteRequest"

	// Load the quoted Stream
	streamService := context.factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByURL(context.session, activity.Object().ID(), &stream); err != nil {

		// A quote of anything but a local Stream is not ours to approve
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading quoted Stream", activity.Object().ID())
	}

	// RULE: A QuoteRequest must be delivered to the inbox of the quoted Stream's author.  This is a
	// 422, as for a Follow: the request is well-formed, but no retry will change where it was sent.
	if !stream.IsAuthor(context.user.UserID) {
		return derp.Validation("Quoted Stream does not belong to the User who owns this inbox", stream.StreamID.Hex(), context.user.UserID.Hex(), derp.WithLocation(location))
	}

	// RULE: Only a Stream that accepts quotes approves one.  The refusal sends no Reject (Ben, 2026-10-29)
	if !stream.AcceptsQuotes() {
		return nil
	}

	// Mastodon embeds the quote post in the request; anything else sends a link to it
	quote := activity.Instrument()

	if quote.IsString() {

		loaded, err := quote.Load()

		if err != nil {
			return derp.Wrap(err, location, "Loading quote post", quote.ID())
		}

		quote = loaded
	}

	// RULE: The quote post must belong to the actor who asked, and must quote this Stream
	if err := inbox_ValidateQuotePost(activity, quote); err != nil {
		return derp.Wrap(err, location, "Validating quote post", quote.ID())
	}

	// Record the approval, which names the QuoteAuthorization
	approval, err := streamService.ApproveQuote(context.session, &stream, quote.ID())

	if err != nil {
		return derp.Wrap(err, location, "Approving quote", stream.StreamID, quote.ID())
	}

	// Send the "Accept" to the quoting actor as a post-commit queue task
	authorizationURL := stream.ActivityPubQuoteAuthorizationURL(approval.CollectionItemID)
	context.factory.Outbox().SendAcceptQuote(context.session, context.user.ActivityPubURL(), activity, authorizationURL)

	// Create a QUOTE notification, from which the author may reject the quote.  A notification
	// failure must not fail the approval, so report-and-continue.
	if err := context.factory.Notification().NotifyQuote(context.session, context.user, activity, quote, &stream); err != nil {
		derp.Report(derp.Wrap(err, location, "Creating quote notification", context.user.UserID))
	}

	// Quote, unquote.
	return nil
}

// inbox_ValidateQuotePost returns an error unless the quote post was written by the actor who sent the
// QuoteRequest, lives on that actor's server, and quotes the object that the QuoteRequest names.
func inbox_ValidateQuotePost(activity streams.Document, quote streams.Document) error {

	const location = "handler.activitypub_user.inbox_ValidateQuotePost"

	actorID := activity.ActorID()
	quoteID := quote.ID()

	// RULE: The quote post must have an ID, which the QuoteAuthorization names as its interactingObject
	if quoteID == "" {
		return derp.Validation("Quote post must have an ID", actorID, derp.WithLocation(location))
	}

	// RULE: An embedded quote post is trusted only from its own origin, which signed the request
	if !activitypub.IsSameOrigin(quoteID, actorID) {
		return derp.Validation("Quote post must share an origin with the requesting actor", quoteID, actorID, derp.WithLocation(location))
	}

	// RULE: Only the author of the quote post may ask to quote
	if quote.AttributedTo().ID() != actorID {
		return derp.Validation("Quote post must be attributed to the requesting actor", quoteID, actorID, derp.WithLocation(location))
	}

	// RULE: The quote post must quote the object that the request names
	if objectID := activity.Object().ID(); quote.Quote().ID() != objectID {
		return derp.Validation("Quote post must quote the requested object", quoteID, objectID, derp.WithLocation(location))
	}

	// All good, chief.
	return nil
}
