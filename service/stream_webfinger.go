package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
)

/******************************************
 * WebFinger Behavior
 ******************************************/

// WebFinger returns the WebFinger resource that describes the Stream with the provided token
func (service *Stream) WebFinger(session data.Session, token string) (digit.Resource, error) {

	const location = "service.Stream.WebFinger"

	// Load the Stream, by token or by StreamID
	stream := model.NewStream()
	if err := service.LoadByToken(session, token, &stream); err != nil {

		// RULE: An unknown token is a 404, not a bad request (RFC 7033 §4.5)
		if derp.IsNotFound(err) {
			return digit.Resource{}, derp.NotFound(location, "Stream not found", token)
		}

		return digit.Resource{}, derp.Wrap(err, location, "Loading Stream", token)
	}

	// Load the Template that defines this Stream's actor
	template, err := service.templateService.Load(stream.TemplateID)

	if err != nil {
		return digit.Resource{}, derp.Wrap(err, location, "Loading Template", stream.TemplateID)
	}

	// RULE: A Stream whose Template defines no actor is not a WebFinger resource
	if template.Actor.IsNil() {
		return digit.Resource{}, derp.NotFound(location, "Stream is not an actor", token)
	}

	hostname := service.Hostname()
	username := stream.ActivityPubUsername()
	streamID := stream.StreamID.Hex()

	// The subject uses the same accessor as the actor document's preferredUsername (see AGENTS.md)
	result := digit.NewResource("acct:" + username + "@" + hostname)

	// The StreamID handle is the durable alias, unless it is already the subject
	if username != streamID {
		result = result.Alias("acct:" + streamID + "@" + hostname)
	}

	// The token URL is an alias, unless the token is the default StreamID
	if stream.Token != streamID {
		result = result.Alias(service.host + "/" + stream.Token)
	}

	// The StreamID URL is the actor id, and always resolves
	result = result.
		Alias(service.host+"/"+streamID).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, stream.ActivityPubURL()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, stream.URL).
		Link(digit.RelationTypeAvatar, model.MimeTypeImage, stream.IconURL)

	return result, nil
}
