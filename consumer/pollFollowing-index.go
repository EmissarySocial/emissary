package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PollFollowing_Index begins the background scheduler that scans all Following records
// according to its own polling frequency
func PollFollowing_Index(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.PollFollowing_Index"

	// Get a list of all Following records that can be polled
	followingService := factory.Following()
	followings, err := followingService.RangePollable(session)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Listing pollable Rollowing records"))
	}

	for following := range followings {

		// RULE: The signature makes a sweep unable to duplicate a task that is still in flight.
		// A poll that returns queue.Error keeps its own task alive through turbine's retry
		// chain (up to ~4h15m), which outlasts the four-hour sweep interval.  See BUG-148.
		postcommit.Publish(session, factory.Queue(), "PollFollowing-Record", mapof.Any{
			"hostname":    factory.Hostname(),
			"userId":      following.UserID.Hex(),
			"followingId": following.FollowingID.Hex(),
		}, queue.WithSignature(pollFollowingSignature(following.FollowingID)))
	}

	return queue.Success()
}

// pollFollowingSignature returns the queue signature that collapses duplicate polls of ONE
// Following record
func pollFollowingSignature(followingID primitive.ObjectID) string {

	// RULE: This must key on the FollowingID and nothing coarser.  A signature built from the
	// UserID would make every Following of one User collapse into a single task, silently
	// leaving all but one of their follows unpolled.
	return "PollFollowing-Record:" + followingID.Hex()
}
