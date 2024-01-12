package ascache

import (
	"net/http"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

type Value struct {

	// Original HTTP Response
	URLs       sliceof.String     `bson:"urls"`                 // One or more URLs used to retrieve this document
	Object     mapof.Any          `bson:"object"`               // Original document, parsed as a map
	HTTPHeader http.Header        `bson:"httpHeader,omitempty"` // HTTP headers that were returned with this document
	Statistics streams.Statistics `bson:"statistics,omitempty"` // Statistics about this document
	Metadata   mapof.Any          `bson:"metadata,omitempty"`   // Metadata about this document

	// Caching Rules
	Published   int64 `bson:"published"`   // Unix epoch seconds when this document was published
	Received    int64 `bson:"received"`    // Unix epoch seconds when this document was received by the cache
	Expires     int64 `bson:"expires"`     // Unix epoch seconds when this document is expired. After this date, it must be revalidated from the source.
	Revalidates int64 `bson:"revalidates"` // Unix epoch seconds when this document should be removed from the cache.
}

func NewValue() Value {
	return Value{
		URLs:       make([]string, 1),
		Object:     make(mapof.Any),
		HTTPHeader: make(http.Header),
		Statistics: streams.NewStatistics(),
		Metadata:   make(mapof.Any),
	}
}

// ShouldRevalidate returns TRUE if the "RevalidatesDate" is in the past.
func (value Value) ShouldRevalidate() bool {
	return value.Revalidates < time.Now().Unix()
}

// calcPublished calculates the date that a document was sent/refreshed by the origin.
// This IS NOT the original create or publish date.
func (value *Value) calcPublished() {

	value.Published = time.Now().Unix()

	// Use the "Date" header, if it exists
	if dateString := value.HTTPHeader.Get(HeaderDate); dateString != "" {
		if date, err := http.ParseTime(dateString); err == nil {
			value.Published = date.Unix()
		}
	}

	// If the "Age" header exists, use it to calculate the original "Published" date
	if ageString := value.HTTPHeader.Get(HeaderAge); ageString != "" {
		if age, err := strconv.ParseInt(ageString, 10, 64); err == nil {
			value.Published = value.Published - age
		}
	}
}

// calcExpires calculates the expiration date for this document.
func (value *Value) calcExpires(cacheControl cacheheader.Header) {

	// If we have a Max-Age value, then use that.
	if cacheControl.MaxAge > 0 {
		value.Expires = value.Published + cacheControl.MaxAge
		return
	}

	// Fall back to (deprecated) Expires header
	if expiresString := value.HTTPHeader.Get(HeaderExpires); expiresString != "" {
		if expires, err := http.ParseTime(expiresString); err == nil {
			value.Expires = expires.Unix()
		}
	}

	// Zero is failure.
	value.Expires = 0
}

// calcRevalidates clculates the date that this document should be revalidated,
// which includes the published date + the "Stale-While-Revalidate" header.
func (value *Value) calcRevalidates(cacheControl cacheheader.Header) {

	// If we have a "Stale-While-Revalidate" header, then use that.
	if cacheControl.StaleWhileRevalidate > 0 {
		value.Revalidates = value.Published + cacheControl.StaleWhileRevalidate
		return
	}

	// Otherwise, items must revalidate as soon as they expire.
	value.Revalidates = value.Expires
}

// calcRelationType calculates the "RelationType" and "RelationHref" metadata for this
// cached document.
func (value *Value) calcRelationType(document streams.Document) {

	// Calculate RelationType
	switch document.Type() {

	// Announce, Like, and Dislike are written straight to the cache.
	case vocab.ActivityTypeAnnounce,
		vocab.ActivityTypeLike,
		vocab.ActivityTypeDislike:

		value.Metadata[PropertyRelationType] = document.Type()
		value.Metadata[PropertyRelationHref] = document.Object().ID()

	// Otherwise, see if this is a "Reply"
	default:
		unwrapped := document.UnwrapActivity()

		if inReplyTo := unwrapped.InReplyTo(); inReplyTo.NotNil() {
			value.Metadata[PropertyRelationType] = RelationTypeReply
			value.Metadata[PropertyRelationHref] = inReplyTo.String()
		}
	}
}

// calcDocumentType sets metadata values for isActor, isObject, and isCollection
func (value *Value) calcDocumentType(document streams.Document) {

	// Set Other Metadata
	switch {

	case document.IsActor():
		value.Metadata[PropertyIsActor] = true

	case document.IsObject():
		value.Metadata[PropertyIsObject] = true

	case document.IsCollection():
		value.Metadata[PropertyIsCollection] = true
	}
}
