package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// bridgyFedBlueskyActor is the Bridgy Fed actor that gates the Bluesky bridge: following it opts the
// User in, blocking it opts them out. Written as a webfinger handle because that is the address
// Bridgy Fed publishes; the Rule and Following services resolve it to its canonical URL themselves.
const bridgyFedBlueskyActor = "@bsky.brid.gy@bsky.brid.gy"

// connectBluesky follows or blocks the Bridgy Fed actor, mirroring this User's Bluesky bridge setting
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
		return derp.Wrap(err, location, "Loading Bluesky configuration", user)
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

	// RULE: Unblock BEFORE following. Leaving the bridge blocks this actor, so a User who re-joins
	// still carries that block -- and a block makes Following.Connect refuse the actor outright.
	// Following first therefore fails before the unblock it depends on ever runs, making the bridge
	// impossible to re-enable once it has been turned off.
	if err := service.ruleService.UnblockActor(session, userID, bridgyFedBlueskyActor); err != nil {
		return derp.Wrap(err, location, "Unblocking Bridgy Fed Actor", userID, connection)
	}

	if _, err := service.followingService.Follow(session, userID, bridgyFedBlueskyActor); err != nil {
		return derp.Wrap(err, location, "Following Bridgy Fed Actor", userID, connection)
	}

	return nil
}

// connectBluesky_unfollow will block the Bridgy Fed Actor (and remove any follows) to leave the bridge
func (service *User) connectBluesky_unfollow(session data.Session, userID primitive.ObjectID, connection *model.Connection) error {

	const location = "service.User.connectBluesky_unfollow"

	if err := service.followingService.Unfollow(session, userID, bridgyFedBlueskyActor); err != nil {
		return derp.Wrap(err, location, "Unfollowing Bridgy Fed Actor", userID, connection)
	}

	if err := service.ruleService.BlockActor(session, userID, bridgyFedBlueskyActor, "Blocking to stop bridge to Bluesky"); err != nil {
		return derp.Wrap(err, location, "Blocking Bridgy Fed Actor", userID, connection)
	}

	return nil
}
