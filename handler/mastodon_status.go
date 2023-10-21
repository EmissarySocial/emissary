package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/relvacode/iso8601"
)

// https://docs.joinmastodon.org/methods/statuses/#create
func mastodon_PostStatus(serverFactory *server.Factory) func(*http.Request, txn.PostStatus) (object.Status, error) {

	const location = "handler.mastodon_PostStatus"
	return func(request *http.Request, transaction txn.PostStatus) (object.Status, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Confirm that the user is Authorized
		authorization, err := getMastodonAuthorization(transaction.Authorization)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Invalid Authorization")
		}

		// Create the stream for the new mastodon "Status"
		stream := model.NewStream()
		stream.ParentID = authorization.UserID
		stream.SocialRole = vocab.ObjectTypeNote
		stream.InReplyTo = transaction.InReplyToID
		stream.Label = transaction.SpoilerText
		stream.Content.Format = model.ContentFormatHTML
		stream.Content.Raw = transaction.Status

		if scheduledAt, err := iso8601.ParseString(transaction.ScheduledAt); err == nil {
			stream.PublishDate = scheduledAt.Unix()
		}

		// Save the stream
		streamService := factory.Stream()
		if err := streamService.Save(&stream, "Created via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error saving stream")
		}

		return streamService.ToToot(&stream)
	}
}

// https://docs.joinmastodon.org/methods/statuses/#get
func mastodon_GetStatus(serverFactory *server.Factory) func(*http.Request, txn.GetStatus) (object.Status, error) {

	const location = "handler.mastodon_GetStatus"

	return func(request *http.Request, transaction txn.GetStatus) (object.Status, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the Stream
		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByURL(transaction.ID, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error loading stream")
		}

		return streamService.ToToot(&stream)
	}
}

// https://docs.joinmastodon.org/methods/statuses/#delete
func mastodon_DeleteStatus(serverFactory *server.Factory) func(*http.Request, txn.DeleteStatus) (object.Status, error) {

	const location = "handler.mastodon_DeleteStatus"

	return func(request *http.Request, transaction txn.DeleteStatus) (object.Status, error) {

		stream := model.NewStream()
		renderer, err := render.NewStreamFromURI(serverFactory, ctx, stream, "delete")

		// Get the factory for this domain
		factory, err := serverFactory.ByDomainName(request.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the Stream
		streamService := factory.Stream()

		if err := streamService.LoadByURL(transaction.ID, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error loading stream")
		}

		streamService.ObjectUserCan()
	}
}
