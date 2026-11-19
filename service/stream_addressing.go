package service

import (
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Addressing returns the `to` and `cc` of an activity that a local actor sends about a Stream,
// naming the audience that the Stream's DefaultAllow grants
func (service *Stream) Addressing(stream *model.Stream, actorType string, actorID primitive.ObjectID) (to []string, cc []string) {

	// Built from DefaultAllow, which every read filters on, so delivery and reads name one audience.
	// Mentions and a reply's parent author are added by the caller (OUTBOX2-OMNIBUS §1.11).
	to = make([]string, 0, 1)
	cc = make([]string, 0, 1)

	// RULE: A public Stream names Public, and copies the sender's followers so that they receive it
	if stream.DefaultAllow.IsAnonymous() {

		to = append(to, vocab.NamespaceActivityStreamsPublic)

		// RULE: An actor with no ID has no followers to name, as a Stream with no author does not
		if actorID.IsZero() {
			return to, cc
		}

		if followersURL := service.locatorService.FollowersURL(actorType, actorID); followersURL != "" {
			cc = append(cc, followersURL)
		}

		return to, cc
	}

	// RULE: Only a User owns circles, so only a User's activity can name them
	if actorType != model.ActorTypeUser {
		return to, cc
	}

	// RULE: An actor with no ID owns no circles
	if actorID.IsZero() {
		return to, cc
	}

	// Name each circle that may view the Stream. Signed-in visitors, site groups, and products
	// name no one, so they receive nothing beyond the people that the Stream mentions.
	circleIDs := stream.CircleIDs()

	for _, permissionID := range stream.DefaultAllow {
		if circleIDs.Contains(permissionID) {
			to = append(to, service.locatorService.CircleURL(actorID, permissionID))
		}
	}

	// Addressed to the right folks. Return to sender? Not this time.
	return to, cc
}

// audience returns the `to` and `cc` of a Stream as its author publishes it: the audience that its
// sharing grants, and every actor that it mentions
func (service *Stream) audience(stream *model.Stream) (to []string, cc []string) {

	to, cc = service.Addressing(stream, model.ActorTypeUser, stream.AttributedTo.UserID)

	// RULE: A mentioned actor must also be ADDRESSED. Mastodon reads mentions from `tag`, but delivery
	// reads addressing alone, so a mentioned account that does not follow the author would get nothing.
	// Only a resolved mention has an actor to address.
	for _, tag := range model.TagsOfType(stream.Tags, vocab.LinkTypeMention) {

		if !tag.IsResolved() {
			continue
		}

		if !slices.Contains(cc, tag.Href) {
			cc = append(cc, tag.Href)
		}
	}

	return to, cc
}

// audienceMap returns a Stream's audience as the `to` and `cc` of an activity
func (service *Stream) audienceMap(stream *model.Stream) mapof.Any {
	to, cc := service.audience(stream)
	return mapof.Any{vocab.PropertyTo: to, vocab.PropertyCC: cc}
}
