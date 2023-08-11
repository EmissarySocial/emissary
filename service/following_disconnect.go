package service

import "github.com/EmissarySocial/emissary/model"

func (service *Following) Disconnect(following *model.Following) {

	switch following.Method {
	case model.FollowMethodActivityPub:
		service.disconnect_ActivityPub(following)

	case model.FollowMethodWebSub:
		service.disconnect_WebSub(following)
	}
}
