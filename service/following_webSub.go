package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/gommon/random"
)

func (service *Following) connect_WebSub(following *model.Following, hub digit.Link) error {

	const location = "service.Following.ConnectWebSub"

	var success string
	var failure string

	// Autocompute the topic.  Use "self" link first, or just the following URL
	self := following.GetLink("rel", model.LinkRelationSelf)
	topicURL := first.String(self.Href, following.URL)

	spew.Dump("connect_WebSub", hub, self, topicURL)

	secret := random.String(32)

	transaction := remote.Post(hub.Href).
		Header("Accept", followingMimeStack).
		Form("hub.mode", "subscribe").
		Form("hub.topic", topicURL).
		Form("hub.callback", service.websubCallbackURL(following)).
		Form("hub.secret", secret).
		Form("hub.lease_seconds", "2582000").
		Response(&success, &failure)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending WebSub subscription request", hub.Href)
	}

	spew.Dump("request sent to hub")

	// Update values in the following object
	following.Method = model.FollowMethodWebSub
	following.URL = topicURL
	following.PollDuration = 30
	following.Secret = secret

	// If we're here, then we have successfully imported the RSS feed.
	// Mark the following as having been polled
	if err := service.SetStatus(following, model.FollowingStatusPending, ""); err != nil {
		spew.Dump(err)
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	spew.Dump("done??")
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

			transaction.Send() // Silent fail is okay here.
		}
	}
}

func (service *Following) websubCallbackURL(following *model.Following) string {
	return service.host + "/.websub/" + following.UserID.Hex() + "/" + following.FollowingID.Hex()
}
