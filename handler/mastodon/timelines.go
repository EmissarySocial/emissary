package mastodon

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// https://docs.joinmastodon.org/methods/timelines/

// https://docs.joinmastodon.org/methods/timelines/#public
func GetTimeline_Public(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Public) ([]object.Status, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetTimeline_Public) ([]object.Status, toot.PageInfo, error) {
		return []object.Status{}, toot.PageInfo{}, nil
	}
}

// https://docs.joinmastodon.org/methods/timelines/#tag
func GetTimeline_Hashtag(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Hashtag) ([]object.Status, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetTimeline_Hashtag) ([]object.Status, toot.PageInfo, error) {
		return []object.Status{}, toot.PageInfo{}, nil
	}
}

// https://docs.joinmastodon.org/methods/timelines/#home
func GetTimeline_Home(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Home) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon.GetTimeline_Home"

	return func(auth model.Authorization, t txn.GetTimeline_Home) ([]object.Status, toot.PageInfo, error) {

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Get NewsItems from the database. NewsItem.GetRank() returns "rank", not
		// CreateDate (see queryExpressionByField), and the DESC sort must match the
		// order getPageInfo assumes (newest first) for its MaxID/MinID to be correct.
		newsFeedService := factory.NewsFeed()
		newsItems, err := newsFeedService.QueryByUserID(session, auth.UserID, queryExpressionByField(t, "rank"), option.SortDesc("rank"), option.MaxRows(pageLimit(t.Limit)))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Retrieving newsItems")
		}

		return newsItemsToToots(factory, session, auth, newsItems), getPageInfo(newsItems), nil
	}
}

// https://docs.joinmastodon.org/methods/timelines/#list
func GetTimeline_List(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_List) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon.GetTimeline_List"

	return func(auth model.Authorization, t txn.GetTimeline_List) ([]object.Status, toot.PageInfo, error) {

		// Parse arguments
		folderID, err := primitive.ObjectIDFromHex(t.ListID)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Invalid ListID")
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Get NewsFeed items from the database. Same rank-vs-createDate note as
		// GetTimeline_Home applies here.
		newsFeedService := factory.NewsFeed()
		criteria := queryExpressionByField(t, "rank").AndEqual("folderId", folderID)

		newsItems, err := newsFeedService.QueryByUserID(session, auth.UserID, criteria, option.SortDesc("rank"), option.MaxRows(pageLimit(t.Limit)))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Retrieving newsItems")
		}

		return newsItemsToToots(factory, session, auth, newsItems), getPageInfo(newsItems), nil
	}
}

// newsItemsToToots converts NewsItems into timeline Statuses. A boosted item
// becomes the booster's own Status wrapping the original post, as Mastodon does.
func newsItemsToToots(factory *service.Factory, session data.Session, auth model.Authorization, newsItems []model.NewsItem) []object.Status {

	client := factory.ActivityStream().UserClient(auth.UserID)
	result := make([]object.Status, len(newsItems))

	for index, newsItem := range newsItems {

		post, booster := newsItemToStatus(client, factory, session, newsItem)

		if booster == nil {
			result[index] = post
			continue
		}

		// RULE: the wrapper needs its own ID -- the client treats a status and its
		// reblog with the same ID as one record. Actions (favourite, etc.) target
		// the inner post, whose ID stays the NewsItemID.
		wrapper := newsItem.Toot()
		wrapper.ID = "b" + wrapper.ID
		wrapper.URI = post.URI + "#announce"
		wrapper.URL = wrapper.URI
		wrapper.Favourited = false
		wrapper.Reblogged = false
		wrapper.Account = *booster
		wrapper.Reblog = &post

		result[index] = wrapper
	}

	return result
}

// newsItemsToPosts converts NewsItems into the Statuses of their original posts,
// with no boost wrapper. It is for endpoints that return the post itself
// (favourites, the response to a favourite).
func newsItemsToPosts(factory *service.Factory, session data.Session, auth model.Authorization, newsItems []model.NewsItem) []object.Status {

	client := factory.ActivityStream().UserClient(auth.UserID)
	result := make([]object.Status, len(newsItems))

	for index, newsItem := range newsItems {
		result[index], _ = newsItemToStatus(client, factory, session, newsItem)
	}

	return result
}

// newsItemToStatus builds the Status for the post behind a NewsItem, live-fetching
// its actor and post document to fill in what NewsItem itself doesn't store. The
// returned booster is the account that surfaced the post, when that was a boost
// and the post's real author could be determined; otherwise it is nil.
//
// RULE (Account): the official app never re-fetches the Account embedded in a
// Status after tapping into it from a timeline post -- it reads straight from
// what it already cached, unlike search or an ID lookup, which always
// re-fetch. A placeholder here sticks at 0 followers forever, not just until
// the next refresh. Falls back to NewsItem.Toot()'s placeholder on a failed
// fetch.
//
// RULE (Author): NewsItem.Origin is whoever led us to the post -- the booster,
// for an ANNOUNCE. The author is named by the post document itself.
//
// RULE (Content): NewsItem never stores the post body -- see NewsItem.Toot().
// This fetch is normally a cache hit, since ingesting the document is how the
// NewsItem came to exist in the first place.
func newsItemToStatus(client streams.Client, factory *service.Factory, session data.Session, newsItem model.NewsItem) (object.Status, *object.Account) {

	status := newsItem.Toot()

	if document, err := client.Load(newsItem.Origin.URL); err == nil {
		status.Account = mapDocumentToAccount(factory, session, document)
	}

	document, err := client.Load(newsItem.URL)

	if err != nil {
		return status, nil
	}

	status.Content = document.Content()
	status.SpoilerText = document.Summary()
	status.Sensitive = status.SpoilerText != ""
	status.MediaAttachments = mapDocumentToMediaAttachments(document)

	if newsItem.Origin.Type != model.OriginTypeAnnounce {
		return status, nil
	}

	authorURL := document.AttributedTo().ID()

	if authorURL == "" {
		authorURL = document.ActorID()
	}

	if authorURL == "" || authorURL == newsItem.Origin.URL {
		return status, nil
	}

	booster := status.Account

	if authorDocument, err := client.Load(authorURL); err == nil {
		status.Account = mapDocumentToAccount(factory, session, authorDocument)
	} else {
		status.Account = model.RemoteActorAccount(authorURL, "", "", time.Time{})
	}

	return status, &booster
}

// mapDocumentToMediaAttachments converts a post document's AS2 "attachment"
// property into Mastodon MediaAttachments. Emissary has no media proxy, so URL
// always points at the origin's own file; PreviewURL only does for images.
func mapDocumentToMediaAttachments(document streams.Document) []object.MediaAttachment {

	result := make([]object.MediaAttachment, 0)

	for attachment := range document.Attachment().Range() {

		// RULE: only a real file (Image/Video/Audio/Document) belongs in
		// media_attachments. A post's "attachment" property can also carry a
		// bare AS2 Link (e.g. a link-preview card) with no url/mediaType/file --
		// skip anything outside this set rather than guessing.
		switch attachment.Type() {
		case vocab.ObjectTypeImage, vocab.ObjectTypeVideo, vocab.ObjectTypeAudio, vocab.ObjectTypeDocument:
			// A real file -- keep going.
		default:
			continue
		}

		url := attachment.URL()

		if url == "" {
			continue
		}

		id := attachment.ID()

		if id == "" {
			id = url
		}

		mediaType := mapAttachmentType(attachment)

		// RULE: preview_url must be a static image -- the client loads it as
		// one. An image's own file doubles as its preview; video/audio have no
		// thumbnail to offer, so leave it empty rather than point at non-image
		// bytes (the client tried to decode a raw .mp4 as an image and crashed).
		previewURL := ""

		if mediaType == "image" {
			previewURL = url
		}

		// RULE: width/height matter, not just cosmetics -- the official app
		// lays out a multi-image post by averaging aspect ratios, and an image
		// missing both drops out of that average; if every image in the post
		// is missing them, it divides by zero and crashes.
		//
		// Read via the raw map, not attachment.Width()/Height(): those route
		// through hannibal's property.NewValue(), which has no int32 case, so
		// they silently return 0 for any document that's round-tripped through
		// Mongo (the driver decodes a BSON int as int32). convert.Int() handles
		// int32 fine, so read the map directly instead.
		meta := map[string]any{}
		attachmentMap := attachment.Map()
		width := convert.Int(attachmentMap["width"])
		height := convert.Int(attachmentMap["height"])

		if width > 0 && height > 0 {
			meta["original"] = map[string]any{
				"width":  width,
				"height": height,
			}
		}

		result = append(result, object.MediaAttachment{
			ID:          id,
			Type:        mediaType,
			URL:         url,
			PreviewURL:  previewURL,
			Description: attachment.Name(),
			Meta:        meta,
		})
	}

	return result
}

// mapAttachmentType derives the Mastodon media type [image|gifv|video|audio] for
// one AS2 attachment. mediaType (a real MIME type) is checked first because it's
// more reliable in practice than the AS2 object type, which some origins misreport
// or omit; the object type is only a fallback.
func mapAttachmentType(attachment streams.Document) string {

	switch {

	case strings.HasPrefix(attachment.MediaType(), "image/"):
		return "image"

	case strings.HasPrefix(attachment.MediaType(), "video/"):
		return "video"

	case strings.HasPrefix(attachment.MediaType(), "audio/"):
		return "audio"
	}

	switch attachment.Type() {

	case vocab.ObjectTypeVideo:
		return "video"

	case vocab.ObjectTypeAudio:
		return "audio"
	}

	return "image"
}
