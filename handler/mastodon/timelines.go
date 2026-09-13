package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
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

		// Get NewsItems from the database
		newsFeedService := factory.NewsFeed()
		newsItems, err := newsFeedService.QueryByUserID(session, auth.UserID, queryExpression(t))

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

		// Get NewsFeed items from the database
		newsFeedService := factory.NewsFeed()
		criteria := queryExpression(t).AndEqual("folderId", folderID)

		newsItems, err := newsFeedService.QueryByUserID(session, auth.UserID, criteria)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Retrieving newsItems")
		}

		return newsItemsToToots(factory, session, auth, newsItems), getPageInfo(newsItems), nil
	}
}

// newsItemsToToots converts NewsItems into Statuses, live-fetching each item's
// actor to fill in what NewsItem itself doesn't store.
//
// RULE: the official app never re-fetches the Account embedded in a
// Status after tapping into it from a timeline post -- it reads straight from
// what it already cached, unlike search or an ID lookup, which always
// re-fetch. A placeholder here sticks at 0 followers forever, not just until
// the next refresh. Falls back to NewsItem.Toot()'s placeholder on a failed
// fetch.
func newsItemsToToots(factory *service.Factory, session data.Session, auth model.Authorization, newsItems []model.NewsItem) []object.Status {

	client := factory.ActivityStream().UserClient(auth.UserID)
	result := make([]object.Status, len(newsItems))

	for index, newsItem := range newsItems {

		status := newsItem.Toot()

		if document, err := client.Load(newsItem.Origin.URL); err == nil {
			status.Account = mapDocumentToAccount(factory, session, document)
		}

		result[index] = status
	}

	return result
}
