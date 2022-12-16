package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
)

func (service *Following) ConnectActivityPub(following *model.Following, link digit.Link) error {
	return derp.NewInternalError("service.Following.ConnectActivityPub", "Not Implemented", following)
}

func (service *Following) DisconnectActivityPub(following *model.Following) {
	// NOOP (for now)
}
