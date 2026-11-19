package service

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// quoteAuthorizationPath is the route, beneath a Stream, that serves the CollectionItems of its Quotes collection
const quoteAuthorizationPath = "/pub/quoteAuthorization/"

// ParseCollectionItem parses the Stream token and the CollectionItemID from a local URL that names one
// CollectionItem.  Today only a QuoteAuthorization URL does.
func (service *Locator) ParseCollectionItem(value string) (string, primitive.ObjectID, error) {

	const location = "service.Locator.ParseCollectionItem"

	// RULE: The URL must name this host.  Compare bare hostnames, as locateObjectFromPath does
	if uri.Hostname(value) != uri.Hostname(service.host) {
		return "", primitive.NilObjectID, derp.NotFound(location, "URL must match host", "host: "+service.host)
	}

	// Parsing discards the query string and the fragment
	parsedURL, err := url.Parse(value)

	if err != nil {
		return "", primitive.NilObjectID, derp.NotFound(location, "URL must be valid")
	}

	// Split the path into the Stream token and the item token
	streamToken, itemToken, found := strings.Cut(strings.TrimPrefix(parsedURL.Path, "/"), quoteAuthorizationPath)

	if !found {
		return "", primitive.NilObjectID, derp.NotFound(location, "URL must contain "+quoteAuthorizationPath)
	}

	// RULE: The Stream token is one path segment, and never a User
	if (streamToken == "") || strings.Contains(streamToken, "/") || strings.HasPrefix(streamToken, "@") {
		return "", primitive.NilObjectID, derp.NotFound(location, "URL must name a Stream")
	}

	// Parse the CollectionItemID
	collectionItemID, err := primitive.ObjectIDFromHex(strings.TrimSuffix(itemToken, "/"))

	if err != nil {
		return "", primitive.NilObjectID, derp.NotFound(location, "URL must name a CollectionItem")
	}

	// Located.
	return streamToken, collectionItemID, nil
}
