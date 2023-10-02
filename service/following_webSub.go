package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/labstack/gommon/random"
)

func (service *Following) connect_WebSub(following *model.Following, hub string) (bool, error) {

	const location = "service.Following.ConnectWebSub"

	var success string
	var failure string

	// Autocompute the topic.  Use "self" link first, or just the following URL
	self := following.GetLink("rel", model.LinkRelationSelf)

	// Update values in the following object
	following.Method = model.FollowMethodWebSub
	following.URL = first.String(self.Href, following.URL)
	following.Secret = random.String(32)
	following.PollDuration = 30

	// Send request to the hub
	transaction := remote.Post(hub).
		Header("Accept", followingMimeStack).
		Form("hub.mode", "subscribe").
		Form("hub.topic", following.URL).
		Form("hub.callback", service.websubCallbackURL(following)).
		Form("hub.secret", following.Secret).
		Form("hub.lease_seconds", "2582000").
		Result(&success).
		Error(&failure)

	if err := transaction.Send(); err != nil {
		return false, derp.Wrap(err, location, "Error sending WebSub subscription request", hub)
	}

	// Success!
	return true, nil
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
