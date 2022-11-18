package activitypub

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func asUserURL(userID primitive.ObjectID) string {
	return "/.activitypub/" + userID.Hex()
}

// asActivityStreamsLink converts a URL into an ActivityStreamsLink
func asActivityStreamsLink(urlString string) (vocab.ActivityStreamsLink, error) {

	// Parse the URL
	urlParsed, err := url.Parse(urlString)

	if err != nil {
		return nil, derp.Wrap(err, "activitypub.asActivityStreamsLink", "Error parsing URL", urlString)
	}

	// Create a new JSON-LD ID property
	idProperty := streams.NewJSONLDIdProperty()
	idProperty.SetIRI(urlParsed)

	// Make an ActivityStreams Link with the ID
	result := streams.NewActivityStreamsLink()
	result.SetJSONLDId(idProperty)

	// OMG, that was painful.
	return result, nil
}

func parseItem(item vocab.Type) (userID primitive.ObjectID, itemType string, itemID primitive.ObjectID, err error) {
	return parseURL(item.GetJSONLDId().GetIRI())
}

// parseURL splits a URL into a list of arguments.
func parseURL(url *url.URL) (userID primitive.ObjectID, itemType string, itemID primitive.ObjectID, err error) {

	const location = "service.activitypub.Database.parseURL"

	path := strings.Split(url.Path, "/")

	if len(path) < 3 {
		err = derp.NewBadRequestError(location, "Invalid path.  Too short.", url)
		return
	}

	if path[0] != "" {
		err = derp.NewBadRequestError(location, "Invalid path.  Must begin with '/'", url)
		return
	}

	if path[1] != ".acivitypub" {
		err = derp.NewBadRequestError(location, "Invalid path. Must begin with '/.activitypub'", url)
		return
	}

	userID, err = primitive.ObjectIDFromHex(path[2])

	if err != nil {
		err = derp.NewBadRequestError(location, "Invalid userID", url, err)
		return
	}

	if len(path) > 3 {

		itemType = path[3]

		if len(path) > 4 {
			itemID, err = primitive.ObjectIDFromHex(path[4])

			if err != nil {
				err = derp.NewBadRequestError(location, "Invalid itemID", url, err)
				return
			}
		}
	}

	return
}
