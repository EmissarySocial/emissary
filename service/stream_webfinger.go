package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
)

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *Stream) WebFinger(token string) (digit.Resource, error) {

	const location = "service.User.WebFinger"

	// Load the stream from the database
	stream := model.NewStream()
	if service.LoadByToken(token, &stream) != nil {
		return digit.Resource{}, derp.BadRequestError(location, "Invalid Token", token)
	}

	// Verify Template and Actor
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return digit.Resource{}, derp.Wrap(err, location, "Invalid Template", stream.TemplateID)
	}

	if template.Actor.IsNil() {
		return digit.Resource{}, derp.BadRequestError(location, "Stream Template does not define an Actor", stream.TemplateID)
	}

	hostname := service.Hostname()

	// Make a WebFinger resource for this Stream.
	result := digit.NewResource("acct:"+stream.StreamID.Hex()+"@"+hostname).
		Alias("acct:"+stream.Token+"@"+hostname).
		Alias(service.host+"/"+stream.Token).
		Alias(service.host+"/"+stream.StreamID.Hex()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, stream.ActivityPubURL()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, stream.URL).
		Link(digit.RelationTypeAvatar, model.MimeTypeImage, stream.IconURL)

	return result, nil
}
