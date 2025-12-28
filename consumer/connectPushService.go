package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
)

// ConnectPushService polls an individual Following record for new post from its outbox (or RSS feed)
func ConnectPushService(factory *service.Factory, session data.Session, user *model.User, following *model.Following, args mapof.Any) queue.Result {

	const location = "consumer.ConnectPushService"

	// RULE: Only connect if both the host and following URL are on the same network.
	// Only local servers can connect to local actors, and only public-facing servers can
	// connect to public-facing actors.
	if dt.IsLocalhost(factory.Host()) == dt.IsLocalhost(following.ProfileURL) {

		// Load the Actor that we're trying to Follow
		activityService := factory.ActivityStream(model.ActorTypeUser, user.UserID)
		actor, err := activityService.Client().Load(following.URL, sherlock.AsActor())

		if err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to load ActivityPub Actor", "url: "+following.URL))
		}

		// Try to connect to ActivityPub actor (if present)
		if inbox := actor.Inbox(); inbox.NotNil() {
			if err := factory.Following().ConnectActivityPub(session, following, &actor); err != nil {
				return queue.Error(derp.Wrap(err, location, "Unable to connect to ActivityPub"))
			}
			return queue.Success()
		}

		// Try to connect to a WebSub hub (if present)
		if hub := actor.Endpoints().Get("websub").String(); hub != "" {
			if err := factory.Following().ConnectWebSub(following, hub); err != nil {
				return queue.Error(derp.Wrap(err, location, "Unable to connect to WebSub"))
			}
			return queue.Success()
		}
	}

	// Fall through means that we can't connect to a push service.
	// Update the status to "POLLING" and report a success because there's nothing more to do.
	if err := factory.Following().SetStatusPolling(session, following); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to update Following status", following))
	}

	// Done
	return queue.Success()
}
