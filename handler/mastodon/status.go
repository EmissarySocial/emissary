package mastodon

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"github.com/relvacode/iso8601"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// https://docs.joinmastodon.org/methods/statuses/#create
func PostStatus(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus) (object.Status, error) {

	const location = "handler.mastodon_PostStatus"
	return func(authorization model.Authorization, transaction txn.PostStatus) (object.Status, error) {

		// RULE: every post made here uses a Template that is readable by anyone, so any other
		// visibility would publish a followers-only or direct post to the world. Refuse it.
		if transaction.Visibility != "" && transaction.Visibility != "public" {
			return object.Status{}, derp.BadRequest(location, "Emissary can only post publicly through this API", transaction.Visibility)
		}

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
		// RULE: InReplyTo's schema requires a real URI (Format: "uri"). transaction.InReplyToID
		// is whatever GetStatus handed this client back as an "id" -- a NewsItem's hex ID, a
		// p_-encoded remote URL, or a Stream's hex ID -- none of which validates as a URI on
		// their own; resolveStatusURL converts it back to the post's actual URL first.
		stream.InReplyTo = resolveStatusURL(factory, session, authorization, transaction.InReplyToID)
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

		// Re-parent any media the client uploaded first (see PostMedia) onto this new Stream.
		if err := attachStatusMedia(factory, session, authorization, &stream, transaction.MediaIDs); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Attaching media")
		}

		indexStatus(factory, session, &stream)

		status := tootStream(factory, session, &stream)
		return status, nil
	}
}

// indexStatus brings the search index in line with a Stream, as the web pipeline's search-index step does.
func indexStatus(factory *service.Factory, session data.Session, stream *model.Stream) {

	// Reported, not returned: the post is already saved, and failing here would invite a duplicate retry
	if err := factory.SearchResult().Sync(session, factory.Stream().SearchResult(stream)); err != nil {
		derp.Report(derp.Wrap(err, "handler.mastodon.indexStatus", "Syncing search index", stream.URL))
	}
}

// https://docs.joinmastodon.org/methods/statuses/#get
//
// t.ID may be an encoded remote URL, a NewsItem's hex ID, or a Stream permalink URL;
// each shape is tried in turn.
func GetStatus(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus) (object.Status, error) {

	const location = "handler.mastodon_GetStatus"

	return func(authorization model.Authorization, t txn.GetStatus) (object.Status, error) {

		if postURL, ok := model.DecodeRemoteStatusID(t.ID); ok {
			return getRemoteStatus(serverFactory, authorization, t.Host, postURL, location)
		}

		if localID, err := primitive.ObjectIDFromHex(t.ID); err == nil {
			return getLocalStatus(serverFactory, authorization, t.Host, localID, location)
		}

		return getStatusByStreamURL(serverFactory, authorization, t.ID, location)
	}
}

// getRemoteStatus resolves a status reached some other way than the feed (a profile's
// posts, a hashtag timeline, a notification) -- its ID is the post's own URL, encoded
// per model.DecodeRemoteStatusID.
func getRemoteStatus(serverFactory *server.Factory, authorization model.Authorization, host string, postURL string, location string) (object.Status, error) {

	factory, session, cancel, err := statusSession(serverFactory, host, location)

	if err != nil {
		return object.Status{}, err
	}

	defer cancel()

	return statusForPostURL(factory, session, authorization, postURL, location)
}

// getLocalStatus resolves a status by a local ID -- a NewsItem in the caller's feed
// (a timeline post) or, failing that, a Stream they own or may view directly.
func getLocalStatus(serverFactory *server.Factory, authorization model.Authorization, host string, id primitive.ObjectID, location string) (object.Status, error) {

	factory, session, cancel, err := statusSession(serverFactory, host, location)

	if err != nil {
		return object.Status{}, err
	}

	defer cancel()

	newsItem := model.NewNewsItem()

	if err := factory.NewsFeed().LoadByID(session, authorization.UserID, id, &newsItem); err == nil {
		return reloadedStatus(factory, session, authorization, newsItem.NewsItemID, location)
	}

	stream := model.NewStream()

	if err := factory.Stream().LoadByID(session, id, &stream); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Loading stream", id)
	}

	if err := userCanStream(factory, session, &authorization, &stream, "view"); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Viewing stream")
	}

	status := tootStream(factory, session, &stream)
	return status, nil
}

// getStatusByStreamURL resolves a status from its Stream's own canonical URL -- the
// shape GetStatus originally expected in every case.
func getStatusByStreamURL(serverFactory *server.Factory, authorization model.Authorization, streamURL string, location string) (object.Status, error) {

	factory, _, stream, err := getStreamFromURL(serverFactory, streamURL)

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Loading stream")
	}

	session, cancel, err := factory.Session(time.Minute)

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Creating session")
	}

	defer cancel()

	if err := userCanStream(factory, session, &authorization, &stream, "view"); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Viewing stream")
	}

	status := tootStream(factory, session, &stream)
	return status, nil
}

// resolveStatusURL converts a status ID this API handed out -- a NewsItem's hex ID, an
// encoded remote URL, or a Stream's hex ID -- back into that post's own URL. An ID this
// can't resolve is returned as-is, on the assumption it was already a URL.
func resolveStatusURL(factory *service.Factory, session data.Session, authorization model.Authorization, id string) string {

	if id == "" {
		return ""
	}

	if postURL, ok := model.DecodeRemoteStatusID(id); ok {
		return postURL
	}

	localID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return id
	}

	newsItem := model.NewNewsItem()

	if err := factory.NewsFeed().LoadByID(session, authorization.UserID, localID, &newsItem); err == nil {
		return newsItem.URL
	}

	stream := model.NewStream()

	if err := factory.Stream().LoadByID(session, localID, &stream); err == nil {
		return stream.ActivityPubURL()
	}

	return id
}

// statusSession resolves the Domain factory for the request's Host and opens a
// session, for GetStatus's ID-based resolution paths (t.Host-scoped, unlike
// getStreamFromURL which derives the domain from the URL itself).
func statusSession(serverFactory *server.Factory, host string, location string) (*service.Factory, data.Session, context.CancelFunc, error) {

	factory, err := serverFactory.ByHostname(host)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Unrecognized Domain")
	}

	session, cancel, err := factory.Session(time.Minute)

	if err != nil {
		return nil, nil, nil, derp.Wrap(err, location, "Creating session")
	}

	return factory, session, cancel, nil
}

// https://docs.joinmastodon.org/methods/statuses/#delete
func DeleteStatus(serverFactory *server.Factory) func(model.Authorization, txn.DeleteStatus) (object.Status, error) {

	const location = "handler.mastodon_DeleteStatus"

	return func(authorization model.Authorization, transaction txn.DeleteStatus) (object.Status, error) {

		// RULE: transaction.ID may be a Stream's hex ID or an older permalink URL, so resolve
		// the domain from transaction.Host rather than parsing it as a URL.
		factory, session, cancel, err := statusSession(serverFactory, transaction.Host, location)

		if err != nil {
			return object.Status{}, err
		}

		defer cancel()

		stream := model.NewStream()

		if err := loadStreamByStatusID(factory, session, transaction.ID, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading stream", transaction.ID)
		}

		// Validate that this user is allowed to delete this Stream.  Deleting is an
		// author-only operation, matching the Mastodon API contract.
		if err := userOwnsStream(&authorization, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Deleting stream")
		}

		// RULE: the real Mastodon API returns the deleted Status itself (its client uses this
		// to offer "delete & redraft" -- restoring the text into a new compose box), not an
		// empty object. Build it before Delete empties the Stream's own fields out from under us.
		status := tootStream(factory, session, &stream)
		streamURL := stream.URL

		if err := factory.Stream().Delete(session, &stream, "Deleted via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Deleting stream")
		}

		if err := factory.SearchResult().DeleteByURL(session, streamURL); err != nil {
			derp.Report(derp.Wrap(err, location, "Removing from search index", streamURL))
		}

		return status, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#context
func GetStatus_Context(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_Context) (object.Context, error) {

	return func(auth model.Authorization, t txn.GetStatus_Context) (object.Context, error) {

		// TODO: HIGH: Implement status contexts via Hannibal

		// RULE: zero-value nil slices marshal to JSON `null`, but the Mastodon client's
		// Codable decoder requires a real (even empty) array for both fields -- a `null`
		// here is a hard decode failure on the client, not a harmless "no thread yet".
		return object.Context{
			Ancestors:   []object.Status{},
			Descendants: []object.Status{},
		}, nil
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

	return func(auth model.Authorization, t txn.PostStatus_Favourite) (object.Status, error) {
		return reactToStatus(serverFactory, t.Host, auth, t.ID, vocab.ActivityTypeLike, "👍", false, "handler.mastodon_PostStatus_Favourite")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unfavourite
func PostStatus_Unfavourite(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unfavourite) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unfavourite) (object.Status, error) {
		return reactToStatus(serverFactory, t.Host, auth, t.ID, vocab.ActivityTypeLike, "", true, "handler.mastodon_PostStatus_Unfavourite")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#boost
func PostStatus_Reblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Reblog) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Reblog) (object.Status, error) {
		return reactToStatus(serverFactory, t.Host, auth, t.ID, vocab.ActivityTypeAnnounce, "", false, "handler.mastodon_PostStatus_Reblog")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unreblog
func PostStatus_Unreblog(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unreblog) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unreblog) (object.Status, error) {
		return reactToStatus(serverFactory, t.Host, auth, t.ID, vocab.ActivityTypeAnnounce, "", true, "handler.mastodon_PostStatus_Unreblog")
	}
}

// reactToStatus sets (or, when undo is true, clears) the caller's response of the given
// type on the post behind a status ID, and returns that post -- a NewsItem in the feed,
// or else the URL encoded in the ID.
func reactToStatus(serverFactory *server.Factory, host string, auth model.Authorization, statusID string, responseType string, content string, undo bool, location string) (object.Status, error) {

	factory, err := serverFactory.ByHostname(host)

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Unrecognized Domain")
	}

	session, cancel, err := factory.Session(time.Minute)

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Creating session")
	}

	defer cancel()

	user := model.NewUser()

	if err := factory.User().LoadByID(session, auth.UserID, &user); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Loading user")
	}

	// Find the post: a NewsItem in the feed, or else the URL encoded in the ID
	message := model.NewNewsItem()
	postURL := ""
	inFeed := false

	switch err := loadNewsItemByStatusID(factory, session, auth.UserID, statusID, &message); {

	case err == nil:
		postURL = message.URL
		inFeed = true

	case derp.IsNotFound(err):

		encodedURL, ok := model.DecodeRemoteStatusID(statusID)

		if !ok {
			return object.Status{}, derp.Wrap(err, location, "Loading message")
		}

		postURL = encodedURL

	default:
		return object.Status{}, derp.Wrap(err, location, "Loading message")
	}

	responseService := factory.Response()

	if undo {
		err = responseService.UnsetResponse(session, &user, postURL, responseType)
	} else {
		err = responseService.SetResponse(session, &user, postURL, responseType, content)
	}

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Saving response")
	}

	if inFeed {
		return reloadedStatus(factory, session, auth, message.NewsItemID, location)
	}

	return statusForPostURL(factory, session, auth, postURL, location)
}

// statusForPostURL builds the Status for a post that has no NewsItem, straight
// from the post document, with the caller's own favourite/boost state filled in.
func statusForPostURL(factory *service.Factory, session data.Session, auth model.Authorization, postURL string, location string) (object.Status, error) {

	client := factory.ActivityStream().UserClient(auth.UserID)
	post, err := client.Load(postURL)

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Loading post", postURL)
	}

	authorURL := post.AttributedTo().ID()

	if authorURL == "" {
		authorURL = post.ActorID()
	}

	account, found := loadAccount(client, factory, session, authorURL)

	if !found {
		account = model.RemoteActorAccount(authorURL, "", "", time.Time{})
	}

	status := documentToStatus(post, account)

	responseService := factory.Response()
	response := model.NewResponse()

	status.Favourited = responseService.LoadByUserAndObject(session, auth.UserID, postURL, vocab.ActivityTypeLike, &response) == nil
	status.Reblogged = responseService.LoadByUserAndObject(session, auth.UserID, postURL, vocab.ActivityTypeAnnounce, &response) == nil
	status.Bookmarked = isBookmarked(factory, session, auth.UserID, postURL)

	return status, nil
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
		return setStatusPinned(serverFactory, auth, t.Host, t.ID, true, "handler.mastodon.PostStatus_Pin")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unpin
func PostStatus_Unpin(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unpin) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unpin) (object.Status, error) {
		return setStatusPinned(serverFactory, auth, t.Host, t.ID, false, "handler.mastodon.PostStatus_Unpin")
	}
}

// setStatusPinned features (pins) or unfeatures one of the caller's own posts. A pin is
// Stream.IsFeatured -- the same flag that fills the user's ActivityPub "featured" collection
// (service.Stream.QueryFeaturedByUser), so pins made here federate the same way.
func setStatusPinned(serverFactory *server.Factory, auth model.Authorization, host string, statusID string, pinned bool, location string) (object.Status, error) {

	factory, session, cancel, err := statusSession(serverFactory, host, location)

	if err != nil {
		return object.Status{}, err
	}

	defer cancel()

	stream := model.NewStream()

	if err := loadStreamByStatusID(factory, session, statusID, &stream); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Loading stream", statusID)
	}

	// RULE: pinning is a write, so it is author-only (see userOwnsStream)
	if err := userOwnsStream(&auth, &stream); err != nil {
		return object.Status{}, derp.Wrap(err, location, "Pinning stream")
	}

	// RULE: the featured collection is read by parentId == userId, so only a post in the
	// caller's own outbox can be pinned (a domain owner is authorized above, not here).
	if stream.ParentID != auth.UserID {
		return object.Status{}, derp.BadRequest(location, "Only your own posts can be pinned")
	}

	if stream.IsFeatured != pinned {

		stream.IsFeatured = pinned

		if err := factory.Stream().Save(session, &stream, "Pinned via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Saving stream")
		}
	}

	status := tootStream(factory, session, &stream)
	return status, nil
}

// loadStreamByStatusID loads the Stream behind a status ID -- the Stream's hex ID (what
// Stream.Toot hands out) or, failing that, its URL.
func loadStreamByStatusID(factory *service.Factory, session data.Session, statusID string, stream *model.Stream) error {

	if streamID, err := primitive.ObjectIDFromHex(statusID); err == nil {
		return factory.Stream().LoadByID(session, streamID, stream)
	}

	return factory.Stream().LoadByURL(session, statusID, stream)
}

// https://docs.joinmastodon.org/methods/statuses/#edit
func PutStatus(serverFactory *server.Factory) func(model.Authorization, txn.PutStatus) (object.Status, error) {

	const location = "handler.mastodon.PutStatus"

	return func(auth model.Authorization, t txn.PutStatus) (object.Status, error) {

		// RULE: t.ID is whatever this API handed the client back for the status (a Stream's
		// hex ID, ordinarily) -- never necessarily its own URL, which LoadByURL alone required
		// and made every edit 404 (see loadStreamByStatusID / DeleteStatus's identical fix).
		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.Status{}, err
		}

		defer cancel()

		streamService := factory.Stream()
		stream := model.NewStream()

		if err := loadStreamByStatusID(factory, session, t.ID, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Loading stream", t.ID)
		}

		// Validate that this user is allowed to edit this Stream.  Editing is an
		// author-only operation, matching the Mastodon API contract.
		if err := userOwnsStream(&auth, &stream); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Editing stream")
		}

		// Edit stream values
		//
		// RULE: rebuild via contentService.New, not just .Raw -- .HTML is what every reader
		// (tootStream, the web page, federation) actually renders.
		contentService := factory.Content()
		stream.Content = contentService.New(stream.Content.Format, t.Status)
		stream.Label = t.SpoilerText
		// t.Sensitive
		// t.Language

		// t.MediaIDs
		// t.Poll info...

		// Save the stream to the database
		if err := streamService.Save(session, &stream, "Edited via Mastodon API"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Saving stream")
		}

		indexStatus(factory, session, &stream)

		status := tootStream(factory, session, &stream)
		return status, nil
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

		// RULE: see PutStatus -- t.ID needs the same ID-shape resolution, not LoadByURL alone.
		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return object.StatusSource{}, err
		}

		defer cancel()

		stream := model.NewStream()

		if err := loadStreamByStatusID(factory, session, t.ID, &stream); err != nil {
			return object.StatusSource{}, derp.Wrap(err, location, "Loading stream", t.ID)
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
