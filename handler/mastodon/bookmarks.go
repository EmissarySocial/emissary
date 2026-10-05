package mastodon

import (
	"net/url"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetBookmarks implements the Mastodon "get bookmarks" endpoint, newest first.
// https://docs.joinmastodon.org/methods/bookmarks/
func GetBookmarks(serverFactory *server.Factory) func(model.Authorization, txn.GetBookmarks) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon.GetBookmarks"

	return func(auth model.Authorization, t txn.GetBookmarks) ([]object.Status, toot.PageInfo, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return nil, toot.PageInfo{}, err
		}

		defer cancel()

		limit := pageLimit(t.Limit)
		bookmarks, err := factory.Bookmark().QueryByUser(session, auth.UserID, queryExpression(t), option.MaxRows(limit))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying bookmarks")
		}

		// Paging cursors come from the bookmarks found, so posts that cannot be loaded do not stall the client
		dates := make([]int64, len(bookmarks))

		for index, bookmark := range bookmarks {
			dates[index] = bookmark.CreateDate
		}

		pageInfo := datePageInfo(dates, limit)

		return bookmarksToStatuses(factory, session, auth, bookmarks), pageInfo, nil
	}
}

// https://docs.joinmastodon.org/methods/statuses/#bookmark
func PostStatus_Bookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Bookmark) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Bookmark) (object.Status, error) {
		return setStatusBookmarked(serverFactory, auth, t.Host, t.ID, true, "handler.mastodon.PostStatus_Bookmark")
	}
}

// https://docs.joinmastodon.org/methods/statuses/#unbookmark
func PostStatus_Unbookmark(serverFactory *server.Factory) func(model.Authorization, txn.PostStatus_Unbookmark) (object.Status, error) {

	return func(auth model.Authorization, t txn.PostStatus_Unbookmark) (object.Status, error) {
		return setStatusBookmarked(serverFactory, auth, t.Host, t.ID, false, "handler.mastodon.PostStatus_Unbookmark")
	}
}

// setStatusBookmarked adds or removes the caller's bookmark on a post. No ownership check is
// needed (unlike pins), but the caller must still be able to see the post.
func setStatusBookmarked(serverFactory *server.Factory, auth model.Authorization, host string, statusID string, bookmarked bool, location string) (object.Status, error) {

	factory, session, cancel, err := statusSession(serverFactory, host, location)

	if err != nil {
		return object.Status{}, err
	}

	defer cancel()

	postURL := resolveStatusURL(factory, session, auth, statusID)

	if parsed, err := url.Parse(postURL); err != nil || parsed.Host == "" {
		return object.Status{}, derp.NotFound(location, "Status not found", statusID)
	}

	status, err := statusForBookmarkURL(factory, session, auth, postURL)

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Loading status", statusID)
	}

	if bookmarked {
		err = factory.Bookmark().Set(session, auth.UserID, postURL)
	} else {
		err = factory.Bookmark().Unset(session, auth.UserID, postURL)
	}

	if err != nil {
		return object.Status{}, derp.Wrap(err, location, "Saving bookmark", statusID)
	}

	status.Bookmarked = bookmarked
	return status, nil
}

// bookmarksToStatuses loads the post behind each bookmark, concurrently.  A post that can no
// longer be loaded (deleted, or its server is unreachable) is left out, as Mastodon does.
func bookmarksToStatuses(factory *service.Factory, session data.Session, auth model.Authorization, bookmarks []model.Bookmark) []object.Status {

	const maxConcurrent = 16

	loaded := make([]*object.Status, len(bookmarks))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, maxConcurrent)

	for index := range bookmarks {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int) {
			defer waitGroup.Done()
			defer func() { <-slots }()

			if status, err := statusForBookmarkURL(factory, session, auth, bookmarks[index].URL); err == nil {
				status.Bookmarked = true
				loaded[index] = &status
			}
		}(index)
	}

	waitGroup.Wait()

	result := make([]object.Status, 0, len(bookmarks))

	for _, status := range loaded {
		if status != nil {
			result = append(result, *status)
		}
	}

	return result
}

// statusForBookmarkURL builds the Status for a post URL: from the caller's feed if it is
// there (so it keeps the ID the client already knows), else from a local Stream the caller
// may view, else by fetching the remote post.
func statusForBookmarkURL(factory *service.Factory, session data.Session, auth model.Authorization, postURL string) (object.Status, error) {

	const location = "handler.mastodon.statusForBookmarkURL"

	newsItem := model.NewNewsItem()

	if err := factory.NewsFeed().LoadByURL(session, auth.UserID, postURL, &newsItem); err == nil {
		return newsItemsToPosts(factory, session, auth, []model.NewsItem{newsItem})[0], nil
	}

	stream := model.NewStream()

	if err := factory.Stream().LoadByURL(session, postURL, &stream); err == nil {

		if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
			return object.Status{}, derp.Wrap(err, location, "Viewing stream")
		}

		status := tootStream(factory, session, &stream)
		status.Bookmarked = isBookmarked(factory, session, auth.UserID, postURL)
		return status, nil
	}

	return statusForPostURL(factory, session, auth, postURL, location)
}

// isBookmarked returns TRUE if the User has bookmarked the post at this URL.
func isBookmarked(factory *service.Factory, session data.Session, userID primitive.ObjectID, postURL string) bool {
	bookmark := model.NewBookmark()
	return factory.Bookmark().LoadByUserAndURL(session, userID, postURL, &bookmark) == nil
}

// markBookmarked sets Bookmarked on every status (matched by its URI) that the User has
// bookmarked, with a single query for the whole list.
func markBookmarked(factory *service.Factory, session data.Session, userID primitive.ObjectID, statuses []object.Status) {

	urls := make([]string, 0, len(statuses))

	for _, status := range statuses {
		if status.URI != "" {
			urls = append(urls, status.URI)
		}
	}

	bookmarked, err := factory.Bookmark().FilterBookmarked(session, userID, urls)

	if err != nil {
		derp.Report(derp.Wrap(err, "handler.mastodon.markBookmarked", "Looking up bookmarks"))
		return
	}

	for index := range statuses {
		statuses[index].Bookmarked = bookmarked[statuses[index].URI]
	}
}
