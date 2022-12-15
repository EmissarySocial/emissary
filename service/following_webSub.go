package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	client "github.com/benpate/websub-client"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/gommon/random"
)

func (service *Following) ConnectWebSub(following *model.Following, link digit.Link) error {

	const location = "service.Following.ConnectWebSub"

	// Update values in the following object
	following.Method = model.FollowMethodWebSub
	following.PollDuration = 30
	following.Data.SetString("secret", random.String(32))

	// Try to connect to the WebSub hub
	c := client.New(service.websubCallbackURL())
	sub, err := c.Subscribe(client.SubscribeOptions{
		Hub:      link.Href,
		Topic:    following.URL,
		Callback: service.websubCallbackURL(),
		Secret:   following.Data.GetString("secret"),
	})

	spew.Dump(sub, err)

	if err != nil {
		return derp.Wrap(err, location, "Error subscribing to WebSub hub", link.Href)
	}

	return nil
}

func (service *Following) DisconnectWebSub(following *model.Following) error {
	const location = "service.Following.DisconnectWebSub"
	return derp.NewInternalError(location, "Not Implemented", following)
}

func (service *Following) ReceiveUpdate() error {
	return nil
}

func (service *Following) websubCallbackURL() string {
	return service.host + "/.websub"
}
