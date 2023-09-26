package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/streams"
)

// connect_ActivityPub attempts to connect to a remote user using ActivityPub.
// It returns (TRUE, nil) if successful.
// If there was an error connecting to the remote server, then it returns (FALSE, error)
// If the remote server does not support ActivityPub, then it returns (FALSE, nil)
func (service *Following) connect_ActivityPub(following *model.Following, actor *streams.Document) (bool, error) {

	const location = "service.Following.connect_ActivityPub"

	// Update the Following record with the remote URL
	following.ProfileURL = actor.ID()
	following.StatusMessage = "Pending ActivityPub connection"

	// Try to get the remote actor (the account that we are following)
	remoteActor, err := service.RemoteActor(following)

	if err != nil {
		return false, derp.Wrap(err, location, "Error getting remote actor", following)
	}

	// Try to get the Actor (with encryption keys)
	localActor, err := service.userService.ActivityPubActor(following.UserID)

	if err != nil {
		return false, derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub follow request
	if err := pub.SendFollow(localActor, service.ActivityPubID(following), remoteActor); err != nil {
		return false, derp.Wrap(err, location, "Error sending follow request", following)
	}

	// Success!
	return true, nil
}

// disconnect_ActivityPub disconnects from an ActivityPub source by sending an "Undo" request
// that references the original "Follow" request per spec.
// https://www.w3.org/TR/activitypub/#undo-activity-outbox
func (service *Following) disconnect_ActivityPub(following *model.Following) error {

	const location = "service.Following.disconnect_ActivityPub"

	// Try to get the local Actor (the user who initiated the follow)
	localActor, err := service.userService.ActivityPubActor(following.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to get the Remote Actor (the account that we are following)
	remoteActor, err := service.RemoteActor(following)

	if err != nil {
		return derp.Wrap(err, location, "Error getting remote actor", following)
	}

	// Try to send the ActivityPub Undo request
	if err := pub.SendUndo(localActor, service.AsJSONLD(following), remoteActor); err != nil {
		return derp.Wrap(err, location, "Error sending follow request", remoteActor.Value())
	}

	return nil
}
