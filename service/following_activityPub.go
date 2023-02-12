package service

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/hannibal/pub"
)

// connect_ActivityPub attempts to connect to a remote user using ActivityPub.
// It returns (TRUE, nil) if successful.
// If there was an error connecting to the remote server, then it returns (FALSE, error)
// If the remote server does not support ActivityPub, then it returns (FALSE, nil)
func (service *Following) connect_ActivityPub(following *model.Following, response *http.Response, buffer *bytes.Buffer) (bool, error) {

	const location = "service.Following.connect_ActivityPub"

	// Search for an ActivityPub link for this resource
	remoteProfile := following.Links.Find(
		digit.NewLink(digit.RelationTypeSelf, model.MimeTypeActivityPub, ""),
	)

	// if no ActivityPub link, then exit.
	if remoteProfile.IsEmpty() {
		return false, nil
	}

	// Try to load the user from the database to use as the ActivityPub "actor"
	user := model.NewUser()
	if err := service.userService.LoadByID(following.UserID, &user); err != nil {
		return false, derp.Wrap(err, location, "Error loading user", following.UserID)
	}

	// Try to get the Actor (with encryption keys)
	actor, err := service.userService.ActivityPubActor(&user)
	if err != nil {
		return false, derp.Wrap(err, location, "Error getting ActivityPub actor", user)
	}

	// Update the "Following" record
	following.Method = model.FollowMethodActivityPub
	following.URL = remoteProfile.Href
	following.Secret = ""
	following.PollDuration = 30

	// Save the "Following" record to the database
	if err := service.SetStatus(following, model.FollowingStatusPending, ""); err != nil {
		return false, derp.Wrap(err, location, "Error saving following", following)
	}

	// Try to send the ActivityPub follow request
	if err := pub.PostFollowRequest(actor, following.FollowingID.Hex(), remoteProfile.Href); err != nil {
		return false, derp.Wrap(err, location, "Error sending follow request", following)
	}

	// Success!
	return true, nil
}

func (service *Following) disconnect_ActivityPub(following *model.Following) {
	// NOOP (for now)
}
