package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/hannibal/pub"
	"github.com/benpate/hannibal/streams"
)

// connect_ActivityPub attempts to connect to a remote user using ActivityPub.
// It returns (TRUE, nil) if successful.
// If there was an error connecting to the remote server, then it returns (FALSE, error)
// If the remote server does not support ActivityPub, then it returns (FALSE, nil)
func (service *Following) connect_ActivityPub(following *model.Following) (bool, error) {

	const location = "service.Following.connect_ActivityPub"

	// Search for an ActivityPub link for this resource
	remoteProfile := following.Links.Find(
		digit.NewLink(digit.RelationTypeSelf, model.MimeTypeActivityPub, ""),
	)

	// if no ActivityPub link, then exit.
	if remoteProfile.IsEmpty() {
		return false, nil
	}

	// Try to get the Actor (with encryption keys)
	actor, err := service.userService.ActivityPubActor(following.UserID)

	if err != nil {
		return false, derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub follow request
	if err := pub.SendFollow(actor, service.ActivityPubID(following), remoteProfile.Href); err != nil {
		return false, derp.Wrap(err, location, "Error sending follow request", following)
	}

	// Success!
	return true, nil
}

func (service *Following) disconnect_ActivityPub(following *model.Following) error {

	const location = "service.Following.disconnect_ActivityPub"

	// Search for an ActivityPub link for this resource
	remoteProfile := following.Links.Find(
		digit.NewLink(digit.RelationTypeSelf, model.MimeTypeActivityPub, ""),
	)

	// if no ActivityPub link, then exit.
	if remoteProfile.IsEmpty() {
		return nil
	}

	// Try to get the Actor (with encryption keys)
	actor, err := service.userService.ActivityPubActor(following.UserID)
	if err != nil {
		return derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub Undo request
	activity := streams.NewDocument(service.AsJSONLD(following), nil)

	if err := pub.SendUndo(actor, activity, following.URL); err != nil {
		return derp.Wrap(err, location, "Error sending follow request", following)
	}

	return nil
}
