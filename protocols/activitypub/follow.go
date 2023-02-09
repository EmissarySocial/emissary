package activitypub

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
)

const RequestStatusSuccess = "SUCCESS"

const RequestStatusPending = "PENDING"

const RequestStatusFailure = "FAILURE"

// PostFollowRequest sends a "Follow" request to the target Actor
// actor: The Actor that is sending the request
// followID: The unique ID of this request
// targetID: The ID of the Actor that is being followed
//
// Returns:
// 1. The status of the request (SUCCESS, PENDING, FAILURE)
// 2. An error, if one occurred
func PostFollowRequest(actor Actor, followID string, targetID string) (string, error) {

	// Build the ActivityStream "Follow" request
	activity := mapof.Any{
		"@context": DefaultContext,
		"id":       followID,
		"type":     ActivityTypeFollow,
		"actor":    actor.ActorID,
		"object":   targetID,
	}

	spew.Dump("ActivityPub..  sending follow request", activity, targetID)

	// Send the request
	result, err := Post(actor, activity, targetID)

	spew.Dump(result, err)

	if err != nil {
		return RequestStatusFailure, derp.Wrap(err, "activitypub.Follow", "Error sending Follow request")
	}

	// Do Something...
	return RequestStatusSuccess, nil
}
