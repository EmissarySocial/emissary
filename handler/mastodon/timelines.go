package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// https://docs.joinmastodon.org/methods/timelines/

// https://docs.joinmastodon.org/methods/timelines/#public
func GetTimeline_Public(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Public) ([]object.Status, error) {

	return func(auth model.Authorization, t txn.GetTimeline_Public) ([]object.Status, error) {

	}
}

// https://docs.joinmastodon.org/methods/timelines/#tag
func GetTimeline_Hashtag(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Hashtag) ([]object.Status, error) {

	return func(auth model.Authorization, t txn.GetTimeline_Hashtag) ([]object.Status, error) {

	}
}

// https://docs.joinmastodon.org/methods/timelines/#home
func GetTimeline_Home(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_Home) ([]object.Status, error) {

	return func(auth model.Authorization, t txn.GetTimeline_Home) ([]object.Status, error) {

	}
}

// https://docs.joinmastodon.org/methods/timelines/#list
func GetTimeline_List(serverFactory *server.Factory) func(model.Authorization, txn.GetTimeline_List) ([]object.Status, error) {

	const location = "handler.mastodon.GetTimeline_List"

	return func(auth model.Authorization, t txn.GetTimeline_List) ([]object.Status, error) {

		// Parse arguments
		folderID, err := primitive.ObjectIDFromHex(t.ListID)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid ListID")
		}

		// Get the factory for this Domain
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid Domain")
		}

		// Get Inbox items from the database
		inboxService := factory.Inbox()
		criteria := queryExpression(t).AndEqual("folderId", folderID)

		messages, err := inboxService.QueryByUserID(auth.UserID, criteria)

		if err != nil {
			return nil, derp.Wrap(err, location, "Error retrieving messages")
		}

		return sliceOfToots[model.Message, object.Status](messages), nil
	}
}
