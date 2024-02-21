package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/rs/zerolog/log"
)

// connect_ActivityPub attempts to connect to a remote user using ActivityPub.
// It returns (TRUE, nil) if successful.
// If there was an error connecting to the remote server, then it returns (FALSE, error)
// If the remote server does not support ActivityPub, then it returns (FALSE, nil)
func (service *Following) connect_ActivityPub(following *model.Following, remoteActor *streams.Document) (bool, error) {

	const location = "service.Following.connect_ActivityPub"

	// Update the Following record with the remote URL
	following.ProfileURL = remoteActor.ID()
	following.StatusMessage = "Pending ActivityPub connection"

	// Try to get the Actor (don't need Following channel)
	localActor, err := service.userService.ActivityPubActor(following.UserID, false)

	if err != nil {
		return false, derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub follow request
	followingURL := service.ActivityPubID(following)
	log.Debug().Str("loc", location).Msg("Sending ActivityPub Follow request to: " + remoteActor.ID())
	localActor.SendFollow(followingURL, remoteActor.ID())

	// Success!
	return true, nil
}

// disconnect_ActivityPub disconnects from an ActivityPub source by sending an "Undo" request
// that references the original "Follow" request per spec.
// https://www.w3.org/TR/activitypub/#undo-activity-outbox
func (service *Following) disconnect_ActivityPub(following *model.Following) error {

	const location = "service.Following.disconnect_ActivityPub"

	// Try to get the local Actor (don't need Following channel)
	actor, err := service.userService.ActivityPubActor(following.UserID, false)

	if err != nil {
		return derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub Undo request
	followMap := service.AsJSONLD(following)
	followDocument := streams.NewDocument(followMap, streams.WithClient(service.activityStreams))
	actor.SendUndo(followDocument)

	return nil
}
