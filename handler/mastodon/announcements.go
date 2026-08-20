package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/announcements/

// GetAnnouncements implements the Mastodon "get announcements" endpoint. Emissary has no announcements, so this is always empty.
func GetAnnouncements(serverFactory *server.Factory) func(model.Authorization, txn.GetAnnouncements) ([]object.Announcement, error) {

	return func(auth model.Authorization, t txn.GetAnnouncements) ([]object.Announcement, error) {
		return []object.Announcement{}, nil
	}
}

// PostAnnouncement_Dismiss implements the Mastodon "dismiss announcement" endpoint as a no-op
func PostAnnouncement_Dismiss(serverFactory *server.Factory) func(model.Authorization, txn.PostAnnouncement_Dismiss) (struct{}, error) {

	return func(auth model.Authorization, t txn.PostAnnouncement_Dismiss) (struct{}, error) {
		return struct{}{}, nil
	}
}

// PutAnnouncement_Reaction implements the Mastodon "add announcement reaction" endpoint as a no-op
func PutAnnouncement_Reaction(serverFactory *server.Factory) func(model.Authorization, txn.PutAnnouncement_Reaction) (struct{}, error) {

	return func(auth model.Authorization, t txn.PutAnnouncement_Reaction) (struct{}, error) {
		return struct{}{}, nil
	}

}

// DeleteAnnouncement_Reaction implements the Mastodon "remove announcement reaction" endpoint as a no-op
func DeleteAnnouncement_Reaction(serverFactory *server.Factory) func(model.Authorization, txn.DeleteAnnouncement_Reaction) (struct{}, error) {

	return func(auth model.Authorization, t txn.DeleteAnnouncement_Reaction) (struct{}, error) {
		return struct{}{}, nil
	}
}
