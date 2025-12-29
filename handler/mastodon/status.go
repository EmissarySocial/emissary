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
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the user from the database
		userSerivce := factory.User()
		user := model.NewUser()

		if err := userSerivce.LoadByID(session, authorization.UserID, &user); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to load user")
		}

		// Create the stream for the new mastodon "Status"
		stream := model.NewStream()
		stream.TemplateID = "outbox-message" // TODO: This should not be hard-coded. Is there some way to look this up?
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
			return object.Status{}, derp.Wrap(err, location, "Unable to save stream")
		}

		// Publish the Stream to the User's outbox
		if err := streamService.Publish(session, &user, &stream, "published", true, false); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error publishing stream")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to load stream")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the template for this stream
		templateService := factory.Template()
		template, err := templateService.Load(stream.TemplateID)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to load template")
		}

		// Validate permissions on this Stream/Template
		allowed, err := factory.Permission().UserCan(session, &authorization, &template, &stream, "view")

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error checking permissions")
		}

		if !allowed {
			return object.Status{}, derp.ForbiddenError(location, "User is not authorized to delete this stream")
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
			return struct{}{}, derp.Wrap(err, location, "Unable to load stream")
		}

		if !stream.IsMyself(authorization.UserID) {
			return struct{}{}, derp.ForbiddenError(location, "User is not authorized to delete this stream")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		if err := streamService.Delete(session, &stream, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to delete stream")
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
		_, _, stream, err := getStreamFromURL(serverFactory, t.ID)

		if err != nil {
			return object.Translation{}, derp.Wrap(err, location, "Unable to load stream")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the User
		userService := factory.User()
		user := model.NewUser()
		if err := userService.LoadByID(session, auth.UserID, &user); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to load user")
		}

		// Load the inbox idem being favorited
		inboxService := factory.Inbox()
		message := model.NewMessage()

		if err := inboxService.LoadByURL(session, auth.UserID, t.ID, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to load message")
		}

		// Create the new response
		responseService := factory.Response()
		response := model.NewResponse()
		response.UserID = auth.UserID
		response.Content = "👍"
		response.Object = message.URL
		response.Type = vocab.ActivityTypeLike
		if err := responseService.Save(session, &response, "Created via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to save response")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to load response")
		}

		// Fall through means a response exists.  Delete it
		if err := responseService.Delete(session, &response, "Deleted via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unable to delete response")
		}

		// Return success
		return response.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#boost
func PostStatus_Reblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Reblog) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Reblog) (object.Status, error) {
		return object.Status{}, derp.NotImplementedError("handler.mastodon.PostStatus_Reblog")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unreblog
func PostStatus_Unreblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unreblog) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unreblog) (object.Status, error) {
		return object.Status{}, derp.NotImplementedError("handler.mastodon.PostStatus_Unreblog")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#bookmark
func PostStatus_Bookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Bookmark) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Bookmark) (object.Status, error) {
		return object.Status{}, derp.NotImplementedError("handler.mastodon.PostStatus_Bookmark")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unbookmark
func PostStatus_Unbookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unbookmark) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unbookmark) (object.Status, error) {
		return object.Status{}, derp.NotImplementedError("handler.mastodon.PostStatus_Unbookmark")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the message from the database
		inboxService := factory.Inbox()
		message := model.NewMessage()

		if err := inboxService.LoadByURL(session, auth.UserID, t.ID, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error retrieving message")
		}

		// Mark the message as Muted
		if err := inboxService.MarkMuted(session, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error muting message")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the message from the database
		inboxService := factory.Inbox()
		message := model.NewMessage()

		if err := inboxService.LoadByURL(session, auth.UserID, t.ID, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error retrieving message")
		}

		// Mark the message as Muted
		if err := inboxService.MarkUnmuted(session, &message); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error muting message")
		}

		return message.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#pin
func PostStatus_Pin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Pin) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Pin) (object.Status, error) {
		return object.Status{}, derp.NotImplementedError("handler.mastodon.PostStatus_Pin")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unpin
func PostStatus_Unpin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unpin) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unpin) (object.Status, error) {
		return object.Status{}, derp.NotImplementedError("handler.mastodon.PostStatus_Unpin")
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
			return object.Status{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the message from the database
		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByURL(session, t.ID, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error muting stream")
		}

		// Validate authorization
		if !stream.IsMyself(auth.UserID) {
			return object.Status{}, derp.UnauthorizedError(location, "User is not authorized to edit this stream", derp.WithForbidden())
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
			return object.Status{}, derp.Wrap(err, location, "Unable to save stream")
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
			return object.StatusSource{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Load the message from the database
		streamService := factory.Stream()
		stream := model.NewStream()

		if err := streamService.LoadByURL(session, t.ID, &stream); err != nil {
			return object.StatusSource{}, derp.Wrap(err, location, "Error muting stream")
		}

		result := object.StatusSource{
			ID:          stream.ActivityPubURL(),
			Text:        stream.Content.Raw,
			SpoilerText: stream.Label,
		}

		return result, nil
	}
}
