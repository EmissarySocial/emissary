package mastodon

import (
	"math"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// GetFavourites implements the Mastodon "get favourites" endpoint: the posts the caller has liked, newest first.
// https://docs.joinmastodon.org/methods/favourites/
func GetFavourites(serverFactory *server.Factory) func(model.Authorization, txn.GetFavourites) ([]object.Status, toot.PageInfo, error) {

	const location = "handler.mastodon_GetFavourites"

	return func(auth model.Authorization, t txn.GetFavourites) ([]object.Status, toot.PageInfo, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Favourites are paged by their create date, which also serves as the cursor
		maxDate := int64(math.MaxInt64)

		if parsed, err := strconv.ParseInt(t.MaxID, 10, 64); err == nil {
			maxDate = parsed
		}

		limit := pageLimit(t.Limit)
		responses, err := factory.Response().QueryByUserAndDate(session, auth.UserID, vocab.ActivityTypeLike, maxDate, int(limit))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying favourites")
		}

		// A Response only stores the post's URL, so look each one up in the news
		// feed. A favourite with no NewsItem has nothing to render; skip it.
		newsFeedService := factory.NewsFeed()
		newsItems := make([]model.NewsItem, 0, len(responses))

		for _, response := range responses {

			newsItem := model.NewNewsItem()

			if err := newsFeedService.LoadByURL(session, auth.UserID, response.Object, &newsItem); err != nil {

				if derp.IsNotFound(err) {
					continue
				}

				return nil, toot.PageInfo{}, derp.Wrap(err, location, "Loading message", response.Object)
			}

			newsItems = append(newsItems, newsItem)
		}

		// Paging cursors come from every favourite found, so ones with no post to show do not stall the client
		dates := make([]int64, len(responses))

		for index, response := range responses {
			dates[index] = response.CreateDate
		}

		pageInfo := datePageInfo(dates, limit)

		return newsItemsToPosts(factory, session, auth, newsItems), pageInfo, nil
	}
}
