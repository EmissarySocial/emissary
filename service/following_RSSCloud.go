package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
)

func (service *Following) connect_RSSCloud(following *model.Following, link digit.Link) error {
	return derp.NewInternalError("service.Following.ConnectRSSCloud", "Not Implemented", following)
}

func (service *Following) disconnect_RSSCloud(following *model.Following) {
	// NO OP (for now)
}
