package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/relvacode/iso8601"
)

// https://docs.joinmastodon.org/methods/statuses/#create
func PostStatus(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus) (object.Status, error) {

	const location = "handler.mastodon_PostStatus"
	return func(authorization model.Authorization, transaction txn.PostStatus) (object.Status, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByHostname(transaction.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the user from the database
		userSerivce := factory.User()
		user := model.NewUser()

		if err := userSerivce.LoadByID(session, authorization.UserID, &user); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading user")
		}

		// Create the stream for the new mastodon "Status"
		stream := model.NewStream()
		// Hard-coded default Template, matching service.Stream.Import (service/stream_import.go).
		// This will be updated when we make a registry of default templates in profiles.
		stream.TemplateID = "outbox-message"
		stream.ParentID = authorization.UserID
		stream.AttributedTo = user.PersonLink()
		stream.SocialRole = vocab.ObjectTypeNote
		stream.InReplyTo = transaction.InReplyToID
		stream.Label = transaction.SpoilerText

		if scheduledAt, err := iso8601.ParseString(transaction.ScheduledAt); err == nil {
			stream.PublishDate = scheduledAt.Unix()
		}

		// Add the content into the stream
		contentService := factory.Content()
		stream.Content = contentService.New(model.ContentFormatHTML, transaction.Status)

		// Save the stream
		streamService := factory.Stream()
		if err := streamService.Save(session, &stream, "Created via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Saving stream")
		}

		// Publish the Stream to the User's outbox
		if err := streamService.Publish(session, &user, &stream, "published", true, false); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Publishing stream")
		}

		return stream.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#get
func GetStatus(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus) (object.Status, error) {

	const location = "handler.mastodon_GetStatus"

	return func(authorization model.Authorization, transaction txn.GetStatus) (object.Status, error) {

		// Get the Stream from the URL
		factory, _, stream, err := getStreamFromURL(serverFactory, transaction.ID)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading stream")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Validate that this user is allowed to view this Stream
		if err := userCanStream(factory, session, &authorization, &stream, "view"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Viewing stream")
		}

		// Return the value
		return stream.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#delete
func DeleteStatus(serverFactory *server.Factory) func(model.Authorization, txn.DeleteStatus) (struct{}, error) {

	const location = "handler.mastodon_DeleteStatus"

	return func(authorization model.Authorization, transaction txn.DeleteStatus) (struct{}, error) {

		factory, streamService, stream, err := getStreamFromURL(serverFactory, transaction.ID)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Loading stream")
		}

		// Validate that this user is allowed to delete this Stream.  Deleting is an
		// author-only operation, matching the Mastodon API contract.
		if err := userOwnsStream(&authorization, &stream); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Deleting stream")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		if err := streamService.Delete(session, &stream, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Deleting stream")
		}

		return struct{}{}, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#context
func GetStatus_Context(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_Context) (object.Context, error) {

	return func(auth model.Authorization, t txn.GetStatus_Context) (object.Context, error) {

		// TODO: HIGH: Implement status contexts via Hannibal
		return object.Context{}, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#translate
func PostStatus_Translate(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Translate) (object.Translation, error) {

	const location = "handler.mastodon.PostStatus_Translate"

	return func(auth model.Authorization, t txn.PostStatus_Translate) (object.Translation, error) {

		// Get the Stream from the URL
		factory, _, stream, err := getStreamFromURL(serverFactory, t.ID)

		if err != nil {
			return object.Translation{}, derp.Wrap(err, location, "Loading stream")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Translation{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Validate that this user is allowed to view this Stream
		if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
			return object.Translation{}, derp.Wrap(err, location, "Viewing stream")
		}

		result := object.Translation{
			Content:                stream.Content.HTML,
			DetectedSourceLanguage: "xx",
			Provider:               "No Translation Available.",
		}

		return result, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#reblogged_by
func GetStatus_RebloggedBy(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_RebloggedBy) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetStatus_RebloggedBy) ([]object.Account, toot.PageInfo, error) {
		return []object.Account{}, toot.PageInfo{}, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#favourited_by
func GetStatus_FavouritedBy(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_FavouritedBy) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetStatus_FavouritedBy) ([]object.Account, toot.PageInfo, error) {
		return []object.Account{}, toot.PageInfo{}, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#favourite
func PostStatus_Favourite(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Favourite) (object.Status, error) {

	const location = "handler.mastodon_PostStatus_Favourite"

	return func(auth model.Authorization, t txn.PostStatus_Favourite) (object.Status, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the User
		userService := factory.User()
		user := model.NewUser()
		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading user")
		}

		// Load the news feed item being favorited
		newsFeedService := factory.NewsFeed()
		message := model.NewNewsItem()

		if err := newsFeedService.LoadByURL(session, auth.UserID, t.ID, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading message")
		}

		// Save the Response via SetResponse, which publishes the activity, keeps Likes and Dislikes
		// mutually exclusive, and makes this endpoint idempotent -- as the Mastodon API requires,
		// since un-favouriting has its own endpoint (see PostStatus_Unfavourite, below).
		responseService := factory.Response()

		if err := responseService.SetResponse(session, &user, message.URL, vocab.ActivityTypeLike, "👍"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Saving response")
		}

		// Read the active Response back, so the caller is returned the record that actually
		// persisted -- which, for a favourite that lost a creation race, is the winner's.
		response := model.NewResponse()

		if err := responseService.LoadByUserAndObject(session, auth.UserID, message.URL, vocab.ActivityTypeLike, &response); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading response")
		}

		return response.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unfavourite
func PostStatus_Unfavourite(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unfavourite) (object.Status, error) {

	const location = "handler.mastodon_PostStatus_Unfavourite"

	return func(auth model.Authorization, t txn.PostStatus_Unfavourite) (object.Status, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Search for the Response in the database
		responseService := factory.Response()
		response := model.NewResponse()

		if err := responseService.LoadByUserAndObject(session, auth.UserID, t.ID, vocab.ActivityTypeLike, &response); err != nil {

			// If the response doesn't exist
			if derp.IsNotFound(err) {
				return response.Toot(), nil
			}

			// Otherwise, return a legitimate error
			return object.Status{}, derp.Wrap(err, location, "Loading response")
		}

		// Fall through means a response exists.  Delete it
		if err := responseService.Delete(session, &response, "Deleted via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Deleting response")
		}

		// Return success
		return response.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#boost
func PostStatus_Reblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Reblog) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Reblog) (object.Status, error) {
		return object.Status{}, derp.NotImplemented("handler.mastodon.PostStatus_Reblog")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unreblog
func PostStatus_Unreblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unreblog) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unreblog) (object.Status, error) {
		return object.Status{}, derp.NotImplemented("handler.mastodon.PostStatus_Unreblog")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#bookmark
func PostStatus_Bookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Bookmark) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Bookmark) (object.Status, error) {
		return object.Status{}, derp.NotImplemented("handler.mastodon.PostStatus_Bookmark")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unbookmark
func PostStatus_Unbookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unbookmark) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unbookmark) (object.Status, error) {
		return object.Status{}, derp.NotImplemented("handler.mastodon.PostStatus_Unbookmark")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#mute
func PostStatus_Mute(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Mute) (object.Status, error) {

	const location = "handler.mastodon_PostStatus_Mute"

	return func(auth model.Authorization, t txn.PostStatus_Mute) (object.Status, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the message from the database
		newsFeedService := factory.NewsFeed()
		message := model.NewNewsItem()

		if err := newsFeedService.LoadByURL(session, auth.UserID, t.ID, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Retrieving message")
		}

		// Mark the message as Muted
		if err := newsFeedService.MarkMuted(session, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Muting message")
		}

		return message.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unmute
func PostStatus_Unmute(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unmute) (object.Status, error) {

	const location = "handler.mastodon.PostStatus_Unmute"

	return func(auth model.Authorization, t txn.PostStatus_Unmute) (object.Status, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the message from the database
		newsFeedService := factory.NewsFeed()
		message := model.NewNewsItem()

		if err := newsFeedService.LoadByURL(session, auth.UserID, t.ID, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Retrieving message")
		}

		// Mark the message as Muted
		if err := newsFeedService.MarkUnmuted(session, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Muting message")
		}

		return message.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#pin
func PostStatus_Pin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Pin) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Pin) (object.Status, error) {
		return object.Status{}, derp.NotImplemented("handler.mastodon.PostStatus_Pin")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unpin
func PostStatus_Unpin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unpin) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unpin) (object.Status, error) {
		return object.Status{}, derp.NotImplemented("handler.mastodon.PostStatus_Unpin")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#edit
func PutStatus(serverFactory *server.Factory) func(model.Authorization, txn.PutStatus) (object.Status, error) {

	const location = "handler.mastodon.PutStatus"

	return func(auth model.Authorization, t txn.PutStatus) (object.Status, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the message from the database
		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByURL(session, t.ID, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading stream")
		}

		// Validate that this user is allowed to edit this Stream.  Editing is an
		// author-only operation, matching the Mastodon API contract.
		if err := userOwnsStream(&auth, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Editing stream")
		}

		// Edit stream values
		stream.Content.Raw = t.Status
		stream.Label = t.SpoilerText
		// t.Sensitive
		// t.Language

		// t.MediaIDs
		// t.Poll info...

		// Save the stream to the database
		if err := streamService.Save(session, &stream, "Edited via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Saving stream")
		}

		return stream.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#history
func GetStatus_History(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_History) ([]object.StatusEdit, error) {

	return func(auth model.Authorization, t txn.GetStatus_History) ([]object.StatusEdit, error) {
		return []object.StatusEdit{}, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#source
func GetStatus_Source(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_Source) (object.StatusSource, error) {

	const location = "handler.mastodon.GetStatus_Source"

	return func(auth model.Authorization, t txn.GetStatus_Source) (object.StatusSource, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return object.StatusSource{}, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.StatusSource{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Load the message from the database
		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByURL(session, t.ID, &stream); err != nil {
			return object.StatusSource{}, derp.Wrap(err, location, "Loading stream")
		}

		// Validate that this user is allowed to view this Stream
		if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
			return object.StatusSource{}, derp.Wrap(err, location, "Viewing stream")
		}

		result := object.StatusSource{
			ID:          stream.ActivityPubURL(),
			Text:        stream.Content.Raw,
			SpoilerText: stream.Label,
		}

		return result, nil
	}
}
