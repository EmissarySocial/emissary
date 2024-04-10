package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

func (service *Following) Disconnect(following *model.Following) {

	switch following.Method {

	case model.FollowingMethodActivityPub:

		if err := service.disconnect_ActivityPub(following); err != nil {
			derp.Report(derp.Wrap(err, "emissary.service.Following.Disconnect", "Error disconnecting from ActivityPub service"))
		}

	case model.FollowingMethodWebSub:
		service.disconnect_WebSub(following)
	}
}
