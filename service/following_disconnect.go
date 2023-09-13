package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

func (service *Following) Disconnect(following *model.Following) {

	switch following.Method {

	case model.FollowMethodActivityPub:

		if err := service.disconnect_ActivityPub(following); err != nil {
			derp.Report(derp.Wrap(err, "emissary.service.Following.Disconnect", "Error disconnecting from ActivityPub service"))
		}

	case model.FollowMethodWebSub:
		service.disconnect_WebSub(following)
	}
}
