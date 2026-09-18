package mastodon

import (
	"strings"
	"sync"
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

	posts, boosters := newsItemsToStatuses(factory, session, auth, newsItems)
	result := make([]object.Status, len(newsItems))

	for index, newsItem := range newsItems {

		post, booster := posts[index], boosters[index]

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

	posts, _ := newsItemsToStatuses(factory, session, auth, newsItems)
	return posts
}

// newsItemsToStatuses runs newsItemToStatus over every NewsItem, concurrently.
//
// RULE: each item costs several remote fetches (actor, post, author, and the
// collections behind an account's counts), so a cold cache makes them the whole
// request. Run in parallel a cold page costs its slowest item, not the sum.
// Accounts are shared across items, so each distinct account is loaded once.
func newsItemsToStatuses(factory *service.Factory, session data.Session, auth model.Authorization, newsItems []model.NewsItem) ([]object.Status, []*object.Account) {

	// A page is at most 40 items, and each spends its time waiting on remote
	// servers, so let a whole page go at once.
	const maxConcurrent = 16

	client := factory.ActivityStream().UserClient(auth.UserID)
	accounts := newAccountMemo()
	statuses := make([]object.Status, len(newsItems))
	boosters := make([]*object.Account, len(newsItems))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, maxConcurrent)

	for index := range newsItems {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int) {
			defer waitGroup.Done()
			defer func() { <-slots }()
			statuses[index], boosters[index] = newsItemToStatus(client, factory, session, accounts, newsItems[index])
		}(index)
	}

	waitGroup.Wait()
	return statuses, boosters
}

// accountMemo loads each distinct account once per request, however many
// concurrent callers ask for it.
type accountMemo struct {
	mutex sync.Mutex
	items map[string]*accountMemoItem
}

type accountMemoItem struct {
	once    sync.Once
	account object.Account
	found   bool
}

func newAccountMemo() *accountMemo {
	return &accountMemo{items: map[string]*accountMemoItem{}}
}

// get returns the account for a URL, calling load only for the first caller
func (memo *accountMemo) get(url string, load func() (object.Account, bool)) (object.Account, bool) {

	memo.mutex.Lock()
	item, ok := memo.items[url]

	if !ok {
		item = &accountMemoItem{}
		memo.items[url] = item
	}

	memo.mutex.Unlock()

	item.once.Do(func() {
		item.account, item.found = load()
	})

	return item.account, item.found
}

// loadAccount fetches an actor and maps it to an Account, reporting false on failure
func loadAccount(client streams.Client, factory *service.Factory, session data.Session, url string) (object.Account, bool) {

	document, err := client.Load(url)

	if err != nil {
		return object.Account{}, false
	}

	return mapDocumentToAccount(factory, session, document), true
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
func newsItemToStatus(client streams.Client, factory *service.Factory, session data.Session, accounts *accountMemo, newsItem model.NewsItem) (object.Status, *object.Account) {

	status := newsItem.Toot()

	if account, found := accounts.get(newsItem.Origin.URL, func() (object.Account, bool) {
		return loadAccount(client, factory, session, newsItem.Origin.URL)
	}); found {
		status.Account = account
	}

	document, err := client.Load(newsItem.URL)

	if err != nil {
		return status, nil
	}

	status.Content = document.Content()
	status.SpoilerText = document.Summary()
	status.Sensitive = status.SpoilerText != ""
	status.MediaAttachments = mapDocumentToMediaAttachments(document)
	status.Tags = mapDocumentToTags(document)

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

	if author, found := accounts.get(authorURL, func() (object.Account, bool) {
		return loadAccount(client, factory, session, authorURL)
	}); found {
		status.Account = author
	} else {
		status.Account = model.RemoteActorAccount(authorURL, "", "", time.Time{})
	}

	return status, &booster
}

// mapDocumentToTags converts the Hashtags in a post document's AS2 "tag" property
// into Mastodon tags. Mentions and emoji share that property but are not hashtags.
func mapDocumentToTags(document streams.Document) []object.StatusTag {

	result := make([]object.StatusTag, 0)

	for tag := range document.Tag().Range() {

		if tag.Type() != vocab.LinkTypeHashtag {
			continue
		}

		// AS2 names carry the leading "#"; Mastodon's do not
		name := strings.TrimPrefix(tag.Name(), "#")
		href := tag.Href()

		if href == "" {
			href = tag.ID()
		}

		if name == "" || href == "" {
			continue
		}

		result = append(result, object.StatusTag{Name: name, URL: href})
	}

	return result
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
