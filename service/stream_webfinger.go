package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/list"
)

/******************************************
 * WebFinger Behavior
 ******************************************/

func (service *Stream) LoadWebFinger(token string) (digit.Resource, error) {
	const location = "service.User.LoadWebFinger"

	switch {

	case domain.HasProtocol(token):
		token = list.Last(token, '@')
		token = list.First(token, '/')

	case strings.HasPrefix(token, "acct:"):
		// Trim prefixes "acct:" and "@"
		token = strings.TrimPrefix(token, "acct:")
		token = strings.TrimPrefix(token, "@")

		// Trim @domain.name suffix if present
		token = strings.TrimSuffix(token, "@"+domain.NameOnly(service.host))

		// Trim path suffix if present
		token = list.First(token, '/')

	default:
		return digit.Resource{}, derp.NewBadRequestError(location, "Invalid token", token)
	}

	// Try to load the user from the database
	stream := model.NewStream()
	if err := service.LoadByToken(token, &stream); err != nil {
		return digit.Resource{}, derp.Wrap(err, location, "Error loading Stream", token)
	}

	// Verify Template and Actor
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return digit.Resource{}, derp.Wrap(err, location, "Invalid Template", stream.TemplateID)
	}

	if template.Actor.IsNil() {
		return digit.Resource{}, derp.NewBadRequestError(location, "Stream Template does not define an Actor", stream.TemplateID)
	}

	// Make a WebFinger resource for this user.
	result := digit.NewResource("acct:"+token+"@"+domain.NameOnly(service.host)).
		Alias(stream.URL).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, stream.ActivityPubURL()).
		// Link(digit.RelationTypeHub, model.MimeTypeJSONFeed, stream.JSONFeedURL()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, stream.URL) //.
		// Link(digit.RelationTypeAvatar, model.MimeTypeImage, stream.ActivityPubIconURL()).
		// Link(digit.RelationTypeSubscribeRequest, "", service.RemoteFollowURL())

	return result, nil
}
