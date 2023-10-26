package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/announcements/

func GetAnnouncements(serverFactory *server.Factory) func(model.Authorization, txn.GetAnnouncements) ([]object.Announcement, error) {

	return func(auth model.Authorization, t txn.GetAnnouncements) ([]object.Announcement, error) {
		return []object.Announcement{}, nil
	}
}

func PostAnnouncement_Dismiss(serverFactory *server.Factory) func(model.Authorization, txn.PostAnnouncement_Dismiss) (struct{}, error) {

	return func(auth model.Authorization, t txn.PostAnnouncement_Dismiss) (struct{}, error) {
		return struct{}{}, nil
	}
}

func PutAnnouncement_Reaction(serverFactory *server.Factory) func(model.Authorization, txn.PutAnnouncement_Reaction) (struct{}, error) {

	return func(auth model.Authorization, t txn.PutAnnouncement_Reaction) (struct{}, error) {
		return struct{}{}, nil
	}

}

func DeleteAnnouncement_Reaction(serverFactory *server.Factory) func(model.Authorization, txn.DeleteAnnouncement_Reaction) (struct{}, error) {

	return func(auth model.Authorization, t txn.DeleteAnnouncement_Reaction) (struct{}, error) {
		return struct{}{}, nil
	}
}
