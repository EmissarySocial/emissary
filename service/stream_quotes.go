package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Quote Posts (FEP-044f)
 *
 * Each quote that a Stream's author approves is one
 * CollectionItem in the Stream's "Quotes" collection,
 * whose URI is the quote post.  The item's ID names
 * the QuoteAuthorization that approves it.
 ******************************************/

// ApproveQuote records an approved quote of this Stream, and returns the CollectionItem that its
// QuoteAuthorization is served from.  Approving the same quote again returns the same item.
func (service *Stream) ApproveQuote(session data.Session, stream *model.Stream, quoteURI string) (model.CollectionItem, error) {

	const location = "service.Stream.ApproveQuote"

	result := model.NewCollectionItem()

	// RULE: An approval must name the quote post that it approves
	if quoteURI == "" {
		return result, derp.BadRequest(location, "Quote URI is required", stream.StreamID)
	}

	// RULE: Only a Stream that accepts quotes may approve one
	if !stream.AcceptsQuotes() {
		return result, derp.Forbidden(location, "Stream does not accept quotes", stream.StreamID)
	}

	// Find (or create) the Stream's Quotes collection
	collection, err := service.collectionService.LoadOrCreateByStream(session, stream, model.CollectionTypeQuotes)

	if err != nil {
		return result, derp.Wrap(err, location, "Loading Quotes collection", stream.StreamID)
	}

	// Add the quote post.  AddItem folds a repeated quote onto its existing item
	if err := service.collectionService.AddItem(session, &collection, quoteURI, stream.URL); err != nil {
		return result, derp.Wrap(err, location, "Adding quote to collection", stream.StreamID, quoteURI)
	}

	// Reload the item, because its ID names the QuoteAuthorization
	if err := service.collectionItemService.LoadByURI(session, collection.CollectionID, quoteURI, &result); err != nil {
		return result, derp.Wrap(err, location, "Loading approved quote", stream.StreamID, quoteURI)
	}

	// I'll allow it.
	return result, nil
}

// LoadQuote loads one approved quote of this Stream.  It returns NotFound whenever the Stream no
// longer accepts quotes, which revokes every QuoteAuthorization it has granted.
func (service *Stream) LoadQuote(session data.Session, stream *model.Stream, collectionItemID primitive.ObjectID, result *model.CollectionItem) error {

	const location = "service.Stream.LoadQuote"

	// RULE: A QuoteAuthorization follows the Stream's current policy, not the one it was granted under
	if !stream.AcceptsQuotes() {
		return derp.NotFound(location, "Stream does not accept quotes", stream.StreamID)
	}

	// Load the CollectionItem from this Stream
	if err := service.collectionItemService.LoadByID(session, stream.StreamID, collectionItemID, result); err != nil {
		return derp.Wrap(err, location, "Loading quote", stream.StreamID, collectionItemID)
	}

	// RULE: Only an item in the Quotes collection is a QuoteAuthorization
	if result.CollectionType != model.CollectionTypeQuotes {
		return derp.NotFound(location, "CollectionItem is not a quote", stream.StreamID, collectionItemID)
	}

	// Quote me on that.
	return nil
}

// LoadQuoteByURI loads the approved quote of one Stream that the quote post at quoteURI made, whatever
// the Stream's current policy.  It returns NotFound when the Stream holds no such quote.
func (service *Stream) LoadQuoteByURI(session data.Session, streamID primitive.ObjectID, quoteURI string, result *model.CollectionItem) error {

	const location = "service.Stream.LoadQuoteByURI"

	// Find the Stream's Quotes collection
	collection := model.NewCollection()

	if err := service.collectionService.LoadByType(session, streamID, model.CollectionTypeQuotes, &collection); err != nil {
		return derp.Wrap(err, location, "Loading Quotes collection", streamID)
	}

	// Find the quote post in it
	if err := service.collectionItemService.LoadByURI(session, collection.CollectionID, quoteURI, result); err != nil {
		return derp.Wrap(err, location, "Loading quote", streamID, quoteURI)
	}

	// Quoth the raven.
	return nil
}

// RejectQuote removes one approved quote of this Stream, whatever the Stream's current policy.
// Deleting the quote's item revokes it.  It does nothing when the Stream has no such quote.
func (service *Stream) RejectQuote(session data.Session, stream *model.Stream, quoteURI string) error {

	const location = "service.Stream.RejectQuote"

	// Find the Stream's Quotes collection
	collection := model.NewCollection()

	if err := service.collectionService.LoadByType(session, stream.StreamID, model.CollectionTypeQuotes, &collection); err != nil {

		// A Stream that was never quoted has nothing to reject
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading Quotes collection", stream.StreamID)
	}

	// Remove the quote, which revokes it
	if err := service.collectionService.RemoveItem(session, &collection, quoteURI); err != nil {
		return derp.Wrap(err, location, "Removing quote", stream.StreamID, quoteURI)
	}

	// Consider it unsaid.
	return nil
}

// RevokeQuote runs the side effects of deleting a quote's CollectionItem: it deletes the author's
// QUOTE notification, and queues a Delete of the QuoteAuthorization for the quoting actor.
// CollectionItem calls it on every delete, so call RejectQuote to reject a quote.
func (service *Stream) RevokeQuote(session data.Session, quote *model.CollectionItem) error {

	const location = "service.Stream.RevokeQuote"

	// Remove the author's notification, so that no Reject button outlives its quote
	if err := service.notificationService.DeleteQuote(session, quote.UserID, quote.ParentID, quote.URI, "Quote rejected"); err != nil {
		return derp.Wrap(err, location, "Deleting quote notification", quote.CollectionItemID)
	}

	// An item that was already gone names no Stream, so there is nothing left to revoke
	if quote.ParentID.IsZero() {
		return nil
	}

	// Load the quoted Stream
	stream := model.NewStream()

	if err := service.LoadByID(session, quote.ParentID, &stream); err != nil {

		// A deleted Stream serves no QuoteAuthorizations, so there is nothing left to revoke
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading quoted Stream", quote.ParentID)
	}

	// Load the author, who signs the Delete
	author := model.NewUser()

	if err := service.userService.LoadByID(session, stream.AttributedTo.UserID, &author); err != nil {
		return derp.Wrap(err, location, "Loading author", stream.StreamID, stream.AttributedTo.UserID)
	}

	// Queue the Delete that tells the quoting actor, and the author's followers, that the approval is
	// withdrawn.  The queue looks up the quoting actor, so this delete never waits on a remote server
	authorization := stream.QuoteAuthorizationJSONLD(*quote)
	service.outboxService.SendDeleteQuoteAuthorization(session, author.ActivityPubURL(), authorization, author.ActivityPubFollowersURL())

	// Take it back.
	return nil
}
