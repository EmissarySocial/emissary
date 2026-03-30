package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *User) connectBluesky(session data.Session, user *model.User) error {

	const location = "service.User.connectBluesky"

	// If the value has not changed, then there's nothing to do
	if user.IsBridgeBluesky.NotChanged() {
		return nil
	}

	// Load the Bluesky connector for this domain.
	connection := model.NewConnection()
	if err := service.connectionService.LoadByProvider(session, model.ConnectionProviderBluesky, &connection); err != nil {

		// If the record is not found, it is just not activated, so return without an error
		if derp.IsNotFound(err) {
			return nil
		}

		// This is a legitimate error.
		return derp.Wrap(err, location, "Unable to load Bluesky configuration", user)
	}

	// If the user has chosen to bridge, then Follow/Unblock the Bridgy Fed Actor
	if user.IsBridgeBluesky.IsTrue() {
		return service.connectBluesky_follow(session, user.UserID, &connection)
	}

	// Otherwise, the user has deactivated the bridge, so Unfollow/Block the Bridgy Fed Actor
	return service.connectBluesky_unfollow(session, user.UserID, &connection)
}

// connectBluesky_follow will follow the Bridgy Fed Actor (and remove any blocks) to join the bridge
func (service *User) connectBluesky_follow(session data.Session, userID primitive.ObjectID, connection *model.Connection) error {

	const location = "service.User.connectBluesky_follow"

	if _, err := service.followingService.Follow(session, userID, "@bsky.brid.gy@bsky.brid.gy"); err != nil {
		return derp.Wrap(err, location, "Unable to follow Bridgy Fed Actor", userID, connection)
	}

	if err := service.ruleService.BlockActor(session, userID, "@bsky.brid.gy@bsky.brid.gy", "Blocking to stop bridge to Bluesky"); err != nil {
		return derp.Wrap(err, location, "Unable to unblock Bridgy Fed Actor", userID, connection)
	}

	return nil
}

// connectBluesky_unfollow will block the Bridgy Fed Actor (and remove any follows) to leave the bridge
func (service *User) connectBluesky_unfollow(session data.Session, userID primitive.ObjectID, connection *model.Connection) error {

	const location = "service.User.connectBluesky_unfollow"

	if err := service.followingService.Unfollow(session, userID, "@bsky.brid.gy@bsky.brid.gy"); err != nil {
		return derp.Wrap(err, location, "Unable to unfollow Bridgy Fed Actor", userID, connection)
	}

	if err := service.ruleService.UnblockActor(session, userID, "@bsky.brid.gy@bsky.brid.gy"); err != nil {
		return derp.Wrap(err, location, "Unable to unblock Bridgy Fed Actor", userID, connection)
	}

	return nil
}
