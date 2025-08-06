package consumer

import (
	"math/rand"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
)

// Start begins the background scheduler that checks each following
// according to its own polling frequency
// TODO: HIGH: Need to make this configurable on a per-physical-server basis so that
// clusters can work together without hammering the Following collection.
func (service *Following) Start(session data.Session) {

	const location = "service.Following.Start"

	// Wait until the service has booted up correctly.
	for service.collection == nil {
		time.Sleep(1 * time.Minute)
	}

	// query the database every minute, looking for following that should be loaded from the web.
	for {

		// If (for some reason) the service collection is still nil, then
		// wait this one out.
		if service.collection == nil {
			continue
		}

		// Get a list of all following that can be polled
		it, err := service.ListPollable(session)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error listing pollable following"))
			continue
		}

		following := model.NewFollowing()

		for it.Next(&following) {
			select {

			// If we're done, we're done.
			case <-service.closed:
				return

			default:

				// Poll each following for new items.
				if err := service.Connect(session, following); err != nil {
					derp.Report(derp.Wrap(err, location, "Error connecting to remote server"))
				}

				// TODO: Reschedule this to run MUCH less frequently
				// if err := service.PurgeInbox(following); err != nil {
				//	derp.Report(derp.Wrap(err, location, "Error purghing inbox"))
				// }
			}

			following = model.NewFollowing()
		}

		// Poll every 4 hours (plus 45 minute jitter)
		time.Sleep(time.Duration(rand.Intn(45)+240) * time.Minute)
	}
}
