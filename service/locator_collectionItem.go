package service

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// quoteAuthorizationPath is the route, beneath a Stream, that serves the CollectionItems of its Quotes collection
const quoteAuthorizationPath = "/pub/quoteAuthorization/"

// CollectionItem loads the CollectionItem that a URL names within one parent's collection of the provided type.
// A local URL that names an item directly is parsed; any other URL is matched against the items' own URIs.
func (service *Locator) CollectionItem(session data.Session, parentID primitive.ObjectID, collectionType string, itemURL string, result *model.CollectionItem) error {

	const location = "service.Locator.CollectionItem"

	// A local URL names the item by its ID
	if _, collectionItemID, err := service.ParseCollectionItem(itemURL); err == nil {

		if err := service.collectionItemService.LoadByID(session, parentID, collectionItemID, result); err != nil {
			return derp.Wrap(err, location, "Loading CollectionItem", parentID, collectionItemID)
		}

		// RULE: The item must be in the collection that the caller asked for
		if result.CollectionType != collectionType {
			return derp.NotFound(location, "CollectionItem is not of the requested type", collectionItemID, collectionType)
		}

		return nil
	}

	// Any other URL is the URI of the thing collected, so find the parent's collection of this type
	collection := model.NewCollection()

	if err := service.collectionService.LoadByType(session, parentID, collectionType, &collection); err != nil {
		return derp.Wrap(err, location, "Loading Collection", parentID, collectionType)
	}

	// Then find the item in it
	if err := service.collectionItemService.LoadByURI(session, collection.CollectionID, itemURL, result); err != nil {
		return derp.Wrap(err, location, "Loading CollectionItem", collection.CollectionID)
	}

	// Gotta catch 'em all.
	return nil
}

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
