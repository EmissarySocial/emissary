package gofed

import (
	"context"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// Followers returns a collection containing the provided Actor's followers.
// This must be the complete collection of followers for that Actor.
func (db Database) Followers(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {

	const location = "gofed.Database.Followers"

	// Get the ownerID from the URL
	ownerID, _, _, err := ParseURL(actorIRI)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing actor IRI", actorIRI)
	}

	it, err := db.followerService.ListActivityPub(ownerID, option.Fields("actor.profileUrl"))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	items := streams.NewActivityStreamsItemsProperty()

	follower := model.NewFollower()

	for it.Next(&follower) {
		items.AppendIRI(follower.Actor.GetURL("profileUrl"))
		follower = model.NewFollower()
	}

	if err := it.Error(); err != nil {
		return nil, derp.Wrap(err, location, "Error iterating database")
	}

	collection := streams.NewActivityStreamsCollection()
	collection.SetActivityStreamsItems(items)

	return collection, nil
}
