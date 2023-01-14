package service

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/digit"
	"github.com/davecgh/go-spew/spew"
)

func (service *Following) connect_ActivityPub(following *model.Following, response *http.Response, buffer *bytes.Buffer) bool {

	spew.Dump("connect_ActivityPub")
	spew.Dump(buffer.String())

	self := following.Links.Find(
		digit.NewLink(
			digit.RelationTypeSelf,
			model.MimeTypeActivityPub,
			"",
		),
	)

	// if no "self"
	if self.IsEmpty() {
		return false
	}

	spew.Dump(self)

	return false
}

func (service *Following) disconnect_ActivityPub(following *model.Following) {
	// NOOP (for now)
}
