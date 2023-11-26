package ascache

import (
	"net/http"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
)

type CachedValue struct {

	// Original HTTP Response
	URI        string      `bson:"uri"`                  // ID/URL of this document
	Original   mapof.Any   `bson:"original"`             // Original document, parsed as a map
	HTTPHeader http.Header `bson:"httpHeader,omitempty"` // HTTP headers that were returned with this document

	// Document Statistics
	Statistics   streams.Statistics `bson:"statistics,omitempty"`   // Statistics about this document
	RelationType string             `bson:"relationType,omitempty"` // Kind of Relationship (Announce, Reply, Like, Dislike)
	RelationHref string             `bson:"relationHref,omitempty"` // HREF of the document that this document is related to.

	// Caching Rules
	Published   int64 `bson:"published"`   // Unix epoch seconds when this document was published
	Received    int64 `bson:"received"`    // Unix epoch seconds when this document was received by the cache
	Expires     int64 `bson:"expires"`     // Unix epoch seconds when this document is expired. After this date, it must be revalidated from the source.
	Revalidates int64 `bson:"revalidates"` // Unix epoch seconds when this document should be removed from the cache.

	// Journal
	journal.Journal `bson:",inline"`
}

func NewCachedValue() CachedValue {
	return CachedValue{
		Original:   make(mapof.Any),
		HTTPHeader: make(http.Header),
		Statistics: streams.NewStatistics(),
	}
}

// ShouldRevalidate returns TRUE if the "RevalidatesDate" is in the past.
func (value CachedValue) ShouldRevalidate() bool {
	return value.Revalidates < time.Now().Unix()
}

// calcPublished calculates the date that a document was sent/refreshed by the origin.
// This IS NOT the original create or publish date.
func (value *CachedValue) calcPublished() {

	value.Published = time.Now().Unix()

	// Use the "Date" header, if it exists
	if dateString := value.HTTPHeader.Get(headerDate); dateString != "" {
		if date, err := http.ParseTime(dateString); err == nil {
			value.Published = date.Unix()
		}
	}

	// If the "Age" header exists, use it to calculate the original "Published" date
	if ageString := value.HTTPHeader.Get(headerAge); ageString != "" {
		if age, err := strconv.ParseInt(ageString, 10, 64); err == nil {
			value.Published = value.Published - age
		}
	}
}

// calcExpires calculates the expiration date for this document.
func (value *CachedValue) calcExpires(cacheControl cacheheader.Header) {

	// If we have a Max-Age value, then use that.
	if cacheControl.MaxAge > 0 {
		value.Expires = value.Published + cacheControl.MaxAge
		return
	}

	// Fall back to (deprecated) Expires header
	if expiresString := value.HTTPHeader.Get(headerExpires); expiresString != "" {
		if expires, err := http.ParseTime(expiresString); err == nil {
			value.Expires = expires.Unix()
		}
	}

	// Zero is failure.
	value.Expires = 0
}

// calcRevalidates clculates the date that this document should be revalidated,
// which includes the published date + the "Stale-While-Revalidate" header.
func (value *CachedValue) calcRevalidates(cacheControl cacheheader.Header) {

	// If we have a "Stale-While-Revalidate" header, then use that.
	if cacheControl.StaleWhileRevalidate > 0 {
		value.Revalidates = value.Published + cacheControl.StaleWhileRevalidate
		return
	}

	// Otherwise, items must revalidate as soon as they expire.
	value.Revalidates = value.Expires
}
