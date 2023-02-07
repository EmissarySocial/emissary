package service

import (
	"bytes"
	"context"
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/rosetta/mapof"
	"github.com/go-fed/activity/streams"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// connect_ActivityPub attempts to connect to a remote user using ActivityPub.
// It returns (TRUE, nil) if successful.
// If there was an error connecting to the remote server, then it returns (FALSE, error)
// If the remote server does not support ActivityPub, then it returns (FALSE, nil)
func (service *Following) connect_ActivityPub(following *model.Following, response *http.Response, buffer *bytes.Buffer) (bool, error) {

	const location = "service.Following.connect_ActivityPub"

	// Search for an ActivityPub link for this resource
	remoteInbox := following.Links.Find(
		digit.NewLink(digit.RelationTypeSelf, model.MimeTypeActivityPub, ""),
	)

	// if no ActivityPub link, then exit.
	if remoteInbox.IsEmpty() {
		return false, nil
	}

	// Calculate the URIs for the user's inbox and outbox
	actorURL := service.host + "/@" + following.UserID.Hex() + "/pub"
	actorOutboxURL := actorURL + "/outbox"
	activityStreamID := actorOutboxURL + "/" + primitive.NewObjectID().Hex()

	// Create a follow message
	jsonLD := mapof.Any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       activityStreamID,
		"type":     "Follow",
		"actor":    actorURL,
		"object":   remoteInbox.Href,
	}

	activityStream, err := streams.ToType(context.TODO(), jsonLD)

	if err != nil {
		return false, derp.Wrap(err, location, "Error converting JSON-LD to Activity Stream", jsonLD)
	}

	// Send the follow message to the user's outbox (which forwards it to the remote inbox )
	actor := service.actorFactory.ActivityPub_Actor()
	actorOutboxURLParsed, _ := url.Parse(actorOutboxURL)
	actor.Send(context.TODO(), actorOutboxURLParsed, activityStream)

	// Mark it as successful (for now)
	// Update values in the following object
	following.Method = model.FollowMethodActivityPub
	following.URL = remoteInbox.Href
	following.Secret = ""
	following.PollDuration = 30

	// "Pending" status means that we're still waiting on the WebSub connection
	if err := service.SetStatus(following, model.FollowingStatusPending, ""); err != nil {
		return false, derp.Wrap(err, location, "Error updating following status", following)
	}

	return true, nil
}

func (service *Following) disconnect_ActivityPub(following *model.Following) {
	// NOOP (for now)
}
