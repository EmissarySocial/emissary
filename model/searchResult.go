package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchResult represents a value in the search index
type SearchResult struct {
	SearchResultID  primitive.ObjectID `bson:"_id"`      // SearchResultID is the unique identifier for a SearchResult.
	StreamID        primitive.ObjectID `bson:"streamId"` // StreamID is the ID of the stream that this SearchResult is associated with.
	URL             string             `bson:"url"`      // URL is the URL of the SearchResult.
	Name            string             `bson:"name"`     // Name is the name of the SearchResult.
	IconURL         string             `bson:"icon"`     // IconURL is the URL of the icon for the SearchResult.
	Tags            []string           `bson:"tags"`     // Tags is a list of tags that are associated with this SearchResult.
	Type            int                `bson:"type"`     // Type is the type of the SearchResult. (ACTOR, OBJECT)
	journal.Journal `bson:",inline"`
}
