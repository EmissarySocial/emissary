package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
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
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Get Inbox items from the database
		inboxService := factory.Inbox()
		messages, err := inboxService.QueryByUserID(session, auth.UserID, queryExpression(t))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Error retrieving messages")
		}

		return getSliceOfToots(messages), getPageInfo(messages), nil
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
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()

		// Get Inbox items from the database
		inboxService := factory.Inbox()
		criteria := queryExpression(t).AndEqual("folderId", folderID)

		messages, err := inboxService.QueryByUserID(session, auth.UserID, criteria)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Error retrieving messages")
		}

		return getSliceOfToots(messages), getPageInfo(messages), nil
	}
}
