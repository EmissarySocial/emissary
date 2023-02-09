package activitypub

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
)

func GetProfile(remoteID string) (mapof.Any, error) {
	return Get(remoteID)
}

func GetInboxURL(remoteID string) (string, error) {
	profile, err := Get(remoteID)
	return profile.GetString("inbox"), err
}

func GetOutboxURL(remoteID string) (string, error) {
	profile, err := Get(remoteID)
	return profile.GetString("outbox"), err
}

func GetFollowersURL(remoteID string) (string, error) {
	profile, err := Get(remoteID)
	return profile.GetString("followers"), err
}

func GetFollowingURL(remoteID string) (string, error) {
	profile, err := Get(remoteID)
	return profile.GetString("following"), err
}

/******************************************
 * Basic HTTP Operations
 ******************************************/

func Get(remoteID string) (mapof.Any, error) {

	// TODO: Some values should be cached internally in this package

	result := mapof.NewAny()

	transaction := remote.Get(remoteID).
		Header("Accept", "application/activity+json").
		Response(&result, nil)

	if err := transaction.Send(); err != nil {
		return result, derp.Wrap(err, "activitypub.GetProfile", "Error getting profile", remoteID)
	}

	return result, nil
}

// Post sends an ActivityStream to a remote ActivityPub service
// actor: The Actor that is sending the request
// activity: The ActivityStream that is being sent
// targetID: The ID of the Actor that will receive the request
//
// Returns:
// 1. The response from the remote service
// 2. An error, if one occurred
func Post(actor Actor, activity mapof.Any, targetID string) (mapof.Any, error) {

	spew.Dump("Post", activity, targetID)

	result := mapof.NewAny()

	// Try to get the source profile that we're going to follow
	target, err := GetProfile(targetID)

	if err != nil {
		return result, derp.Wrap(err, "activitypub.Follow", "Error getting source profile", targetID)
	}

	// Try to get the actor's inbox from the actor ActivityStream.
	// TODO: LOW: Is there a better / more reliable way to do this?
	inbox := target.GetString("inbox")

	if inbox == "" {
		return result, derp.New(500, "activitypub.Follow", "Unable to find 'inbox' in target profile", targetID, target)
	}

	transaction := remote.Post(inbox).
		Accept("application/activity+json").
		ContentType("application/activity+json").
		Use(RequestSignature(actor)).
		JSON(activity).
		Response(&result, nil)

	if err := transaction.Send(); err != nil {
		return result, derp.Wrap(err, "activitypub.Follow", "Error sending Follow request", inbox)
	}

	return result, nil
}
