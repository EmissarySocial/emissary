package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// PollFollowing_Index begins the background scheduler that scans all Following records
// according to its own polling frequency
func PollFollowing_Index(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.PollFollowing_Index"

	// Get a list of all Following records that can be polled
	followingService := factory.Following()
	followings, err := followingService.RangePollable(session)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error listing pollable Rollowing records"))
	}

	for following := range followings {

		factory.Queue().NewTask("PollFollowing-Record", mapof.Any{
			"host":        args.GetString("host"),
			"userId":      following.UserID.Hex(),
			"followingId": following.FollowingID.Hex(),
		})
	}

	return queue.Success()
}
