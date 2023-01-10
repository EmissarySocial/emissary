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

// Following returns a collection containing the provided Actor's following.
// This must be the complete collection of following for that Actor.
func (db Database) Following(c context.Context, actorIRI *url.URL) (vocab.ActivityStreamsCollection, error) {

	const location = "gofed.Database.Following"

	// Get the ownerID from the URL
	ownerID, _, _, err := ParsePath(actorIRI)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing actor IRI", actorIRI)
	}

	it, err := db.followingService.ListActivityPub(ownerID, option.Fields("url"))

	if err != nil {
		return nil, derp.Wrap(err, location, "Error querying database")
	}

	items := streams.NewActivityStreamsItemsProperty()

	following := model.NewFollowing()

	for it.Next(&following) {
		followingURL, _ := url.Parse(following.URL)
		items.AppendIRI(followingURL)
		following = model.NewFollowing()
	}

	if err := it.Error(); err != nil {
		return nil, derp.Wrap(err, location, "Error iterating database")
	}

	collection := streams.NewActivityStreamsCollection()
	collection.SetActivityStreamsItems(items)

	return collection, nil
}
