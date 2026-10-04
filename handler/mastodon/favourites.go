package mastodon

import (
	"math"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/favourites/
//
// The API returns a bare []Status with no PageInfo, so there is no Link header
// to page with; this returns the newest page of favourites only.
func GetFavourites(serverFactory *server.Factory) func(model.Authorization, txn.GetFavourites) ([]object.Status, error) {

	const location = "handler.mastodon_GetFavourites"

	return func(auth model.Authorization, t txn.GetFavourites) ([]object.Status, error) {

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		responses, err := factory.Response().QueryByUserAndDate(session, auth.UserID, vocab.ActivityTypeLike, math.MaxInt64, int(pageLimit(t.Limit)))

		if err != nil {
			return nil, derp.Wrap(err, location, "Querying favourites")
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

				return nil, derp.Wrap(err, location, "Loading message", response.Object)
			}

			newsItems = append(newsItems, newsItem)
		}

		return newsItemsToPosts(factory, session, auth, newsItems), nil
	}
}
