package service

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Unfollow guarantees that the user is no longer following the specified URL.  If the user is already following this actor,
// then the Following record is disconnected. Otherwise, no action is taken.
func (service *Following) Unfollow(session data.Session, userID primitive.ObjectID, actorID string) error {

	const location = "service.Following.Unfollow"
	spew.Dump(location, "Attempting to unfollow URL", actorID)

	// If the actor ID is not already a valid URL, it's probably a username/handle,
	// so try to resolve it into a URL using Sherlock/WebFinger.
	if _, err := url.Parse(actorID); err != nil {

		// Look up the Actor from the Activity service
		actor, err := service.activityService.GetActor(actorID)

		if err != nil {
			return derp.Wrap(err, location, "Unable to find Actor for URL", actorID)
		}

		actorID = actor.ID()
	}

	// Try to load the existing Following record for this user and URL
	following := model.NewFollowing()
	err := service.LoadByURL(session, userID, actorID, &following)

	if err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Unable to load following for user and URL", userID, actorID)
	}

	// Disconnect the Following record
	return service.Delete(session, &following, "")
}

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
	client := service.activityService.UserClient(following.UserID)
	followDocument := streams.NewDocument(followMap, streams.WithClient(client))
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
				derp.Report(derp.Wrap(err, "service.Following.DisconnectWebSub", "Unable to send WebSub unsubscribe request", link.Href))
			}
		}
	}
}

func (service *Following) websubCallbackURL(following *model.Following) string {
	return service.host + "/.websub/" + following.UserID.Hex() + "/" + following.FollowingID.Hex()
}
