package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// Start begins the background scheduler that checks each following
// according to its own polling frequency
// TODO: HIGH: Need to make this configurable on a per-physical-server basis so that
// clusters can work together without hammering the Following collection.
func PollFollowing(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.PollFollowing"

	// Get a list of all Following records that can be polled
	followingService := factory.Following()
	followings, err := followingService.RangePollable(session)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error listing pollable following"))
	}

	for following := range followings {

		// Poll each following for new items.
		if err := followingService.Connect(session, following); err != nil {
			derp.Report(derp.Wrap(err, location, "Error connecting to remote server"))
		}
	}

	return queue.Success()
}
