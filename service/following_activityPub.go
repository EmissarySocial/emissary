package service

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/protocols/activitypub"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/davecgh/go-spew/spew"
)

// connect_ActivityPub attempts to connect to a remote user using ActivityPub.
// It returns (TRUE, nil) if successful.
// If there was an error connecting to the remote server, then it returns (FALSE, error)
// If the remote server does not support ActivityPub, then it returns (FALSE, nil)
func (service *Following) connect_ActivityPub(following *model.Following, response *http.Response, buffer *bytes.Buffer) (bool, error) {

	const location = "service.Following.connect_ActivityPub"

	spew.Dump("connect_ActivityPub", following)

	// Search for an ActivityPub link for this resource
	remoteProfile := following.Links.Find(
		digit.NewLink(digit.RelationTypeSelf, model.MimeTypeActivityPub, ""),
	)

	spew.Dump(remoteProfile)

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
	actor, err := service.ActivityPubActor(&user)
	if err != nil {
		return false, derp.Wrap(err, location, "Error getting ActivityPub actor", user)
	}

	// Try to send the ActivityPub follow request
	status, err := activitypub.PostFollowRequest(actor, following.FollowingID.Hex(), remoteProfile.Href)
	if err != nil {
		return false, derp.Wrap(err, location, "Error sending follow request", following)
	}

	// Update the "Following" record
	following.Status = status
	following.StatusMessage = ""
	following.Method = model.FollowMethodActivityPub
	following.URL = remoteProfile.Href
	following.Secret = ""
	following.PollDuration = 30

	// Save the "Following" record to the database
	if err := service.Save(following, "Subscribed to ActivityPub"); err != nil {
		return false, derp.Wrap(err, location, "Error saving following", following)
	}

	// Success!
	return true, nil
}

func (service *Following) disconnect_ActivityPub(following *model.Following) {
	// NOOP (for now)
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided user.
func (service *Following) ActivityPubActor(user *model.User) (activitypub.Actor, error) {

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByID(user.UserID, &encryptionKey); err != nil {
		return activitypub.Actor{}, derp.Wrap(err, "service.Following.ActivityPubActor", "Error loading encryption key", user.UserID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return activitypub.Actor{}, derp.Wrap(err, "service.Following.ActivityPubActor", "Error extracting private key", encryptionKey)
	}

	// Return the ActivityPub Actor
	return activitypub.Actor{
		ActorID:     user.ActivityPubURL(),
		PublicKeyID: user.ActivityPubPublicKeyURL(),
		PublicKey:   privateKey.PublicKey,
		PrivateKey:  privateKey,
	}, nil
}
