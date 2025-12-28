package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
)

func (service *Following) Disconnect(session data.Session, following *model.Following) {

	switch following.Method {

	case model.FollowingMethodActivityPub:

		if err := service.disconnect_ActivityPub(session, following); err != nil {
			derp.Report(derp.Wrap(err, "emissary.service.Following.Disconnect", "Error disconnecting from ActivityPub service"))
		}

	case model.FollowingMethodWebSub:
		service.disconnect_WebSub(following)
	}
}

// disconnect_ActivityPub disconnects from an ActivityPub source by sending an "Undo" request
// that references the original "Follow" request per spec.
// https://www.w3.org/TR/activitypub/#undo-activity-outbox
func (service *Following) disconnect_ActivityPub(session data.Session, following *model.Following) error {

	const location = "service.Following.disconnect_ActivityPub"

	// Try to get the local Actor (don't need Following channel)
	actor, err := service.userService.ActivityPubActor(session, following.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Error getting ActivityPub actor", following.UserID)
	}

	// Try to send the ActivityPub Undo request
	followMap := service.AsJSONLD(following)
	activityService := service.factory.ActivityStream(model.ActorTypeUser, following.UserID)
	followDocument := streams.NewDocument(followMap, streams.WithClient(activityService.Client()))
	actor.SendUndo(followDocument)

	return nil
}

func (service *Following) disconnect_WebSub(following *model.Following) {

	// Find the "hub" link for this following
	for _, link := range following.Links {
		if link.RelationType == "hub" {

			transaction := remote.Post(link.Href).
				Form("hub.mode", "unsubscribe").
				Form("hub.topic", following.URL).
				Form("hub.callback", service.websubCallbackURL(following))

			if err := transaction.Send(); err != nil {
				derp.Report(derp.Wrap(err, "service.Following.DisconnectWebSub", "Error sending WebSub unsubscribe request", link.Href))
			}
		}
	}
}

func (service *Following) websubCallbackURL(following *model.Following) string {
	return service.host + "/.websub/" + following.UserID.Hex() + "/" + following.FollowingID.Hex()
}
