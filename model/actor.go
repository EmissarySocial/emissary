package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

type Actor struct {
	SocialRole         string `json:"social-role"          bson:"socialRole"`         // Actor Role to use for this Template in social integrations (Person, Organization, Application, etc.)
	RSS                bool   `json:"rss"                  bson:"rss"`                // If TRUE, Generate RSS/Atom/JSONFeed/WebSub endpoints for this actor and its children
	BoostInbox         bool   `json:"boost-inbox"          bson:"boostInbox"`         // If TRUE, Broadcast all events sent to this Stream to all Followers
	BoostFollowersOnly bool   `json:"boost-followers-only" bson:"boostFollowersOnly"` // If TRUE, Broadcast messages from Followers only (not from other sources)
	BoostChildren      bool   `json:"boost-children"       bson:"boostChildren"`      // If TRUE, Broadcast add/update/delete events on child Streams to Followers
	PublishFollowers   bool   `json:"publish-followers"    bson:"publishFollowers"`   // If TRUE, Follower list is published via ActivityPub
}

// IsNull returns TRUE if this actor is nil (or undefined)
func (actor Actor) IsNil() bool {
	return actor.SocialRole == ""
}

// NotNil returns TRUE if this actor has been defined (and should be executed).
func (actor Actor) NotNil() bool {
	return !actor.IsNil()
}

func (actor Actor) JSONLD(stream *Stream) mapof.Any {

	if actor.IsNil() {
		return mapof.Any{}
	}

	permalink := stream.Permalink()

	result := mapof.Any{
		vocab.AtContext:       vocab.ContextTypeActivityStreams,
		vocab.PropertyType:    actor.SocialRole,
		vocab.PropertyID:      stream.Permalink(),
		vocab.PropertyName:    stream.Label,
		vocab.PropertySummary: stream.Summary,
		vocab.PropertyInbox:   permalink + "/inbox",
		vocab.PropertyOutbox:  permalink + "/outbox",
	}

	if actor.PublishFollowers {
		result[vocab.PropertyFollowers] = permalink + "/followers"
	}

	return result
}
