package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/relvacode/iso8601"
)

// https://docs.joinmastodon.org/methods/statuses/#create
func PostStatus(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus) (object.Status, error) {

	const location = "handler.mastodon_PostStatus"
	return func(authorization model.Authorization, transaction txn.PostStatus) (object.Status, error) {

		// Get the factory for this domain
		factory, err := serverFactory.ByDomainName(transaction.Host)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
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

		// Verify user permissions
		streamService := factory.Stream()
		if err := streamService.UserCan(&authorization, &stream, "create"); err != nil {
			return object.Status{}, derp.New(derp.CodeForbiddenError, "User is not authorized to delete this stream", location)
		}

		// Save the stream
		if err := streamService.Save(&stream, "Created via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error saving stream")
		}

		return stream.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#get
func GetStatus(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus) (object.Status, error) {

	const location = "handler.mastodon_GetStatus"

	return func(authorization model.Authorization, transaction txn.GetStatus) (object.Status, error) {

		// Get the Stream from the URL
		stream, streamService, err := getStreamFromURL(serverFactory, transaction.ID)

		if err != nil {
			return object.Status{}, derp.Wrap(err, location, "Error loading stream")
		}

		// Validate permissions
		if err := streamService.UserCan(&authorization, &stream, "view"); err != nil {
			return object.Status{}, derp.New(derp.CodeForbiddenError, "User is not authorized to delete this stream", location)
		}

		// Return the value
		return stream.Toot(), nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#delete
func DeleteStatus(serverFactory *server.Factory) func(model.Authorization, txn.DeleteStatus) (struct{}, error) {

	const location = "handler.mastodon_DeleteStatus"

	return func(authorization model.Authorization, transaction txn.DeleteStatus) (struct{}, error) {

		stream, streamService, err := getStreamFromURL(serverFactory, transaction.ID)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error loading stream")
		}

		if err := streamService.UserCan(&authorization, &stream, "delete"); err != nil {
			return struct{}{}, derp.New(derp.CodeForbiddenError, "User is not authorized to delete this stream", location)
		}

		if err := streamService.Delete(&stream, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error deleting stream")
		}

		return struct{}{}, nil
	}
}

func GetStatus_Context(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_Context) (object.Context, error) {

	return func(model.Authorization, txn.GetStatus_Context) (object.Context, error) {

	}
}

func PostStatus_Translate(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Translate) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Translate) (object.Status, error) {

	}
}

func GetStatus_RebloggedBy(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_RebloggedBy) ([]object.Account, error) {

	return func(model.Authorization, txn.GetStatus_RebloggedBy) ([]object.Account, error) {

	}
}

func GetStatus_FavouritedBy(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_FavouritedBy) ([]object.Account, error) {

	return func(model.Authorization, txn.GetStatus_FavouritedBy) ([]object.Account, error) {

	}
}

func PostStatus_Favourite(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Favourite) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Favourite) (object.Status, error) {

	}
}

func PostStatus_Unfavourite(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unfavourite) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Unfavourite) (object.Status, error) {

	}
}

func PostStatus_Reblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Reblog) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Reblog) (object.Status, error) {

	}
}

func PostStatus_Unreblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unreblog) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Unreblog) (object.Status, error) {

	}
}

func PostStatus_Bookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Bookmark) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Bookmark) (object.Status, error) {

	}
}

func PostStatus_Unbookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unbookmark) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Unbookmark) (object.Status, error) {

	}
}

func PostStatus_Mute(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Mute) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Mute) (object.Status, error) {

	}
}

func PostStatus_Unmute(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unmute) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Unmute) (object.Status, error) {

	}
}

func PostStatus_Pin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Pin) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Pin) (object.Status, error) {

	}
}

func PostStatus_Unpin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unpin) (object.Status, error) {

	return func(model.Authorization, txn.PostStatus_Unpin) (object.Status, error) {

	}
}

func PutStatus(serverFactory *server.Factory) func(model.Authorization, txn.PutStatus) (object.Status, error) {

	return func(model.Authorization, txn.PutStatus) (object.Status, error) {

	}
}

func GetStatus_History(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_History) ([]object.StatusEdit, error) {

	return func(model.Authorization, txn.GetStatus_History) ([]object.StatusEdit, error) {

	}
}

func GetStatus_Source(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_Source) (object.StatusSource, error) {

	return func(model.Authorization, txn.GetStatus_Source) (object.StatusSource, error) {

	}
}
