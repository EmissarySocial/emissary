package ascache

import (
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Value struct {
	ValueID primitive.ObjectID `bson:"_id"`

	// Original HTTP Response
	URLs     sliceof.String   `bson:"urls"`     // One or more URLs used to retrieve this document
	Object   mapof.Any        `bson:"object"`   // Original document, parsed as a map
	Metadata streams.Metadata `bson:"metadata"` // Metadata about this document

	// Caching Rules
	HTTPHeader  http.Header `bson:"httpHeader"`  // HTTP headers that were returned with this document
	Published   int64       `bson:"published"`   // Unix epoch seconds when this document was published
	Received    int64       `bson:"received"`    // Unix epoch seconds when this document was received by the cache
	Expires     int64       `bson:"expires"`     // Unix epoch seconds when this document is expired. After this date, it must be revalidated from the source.
	Revalidates int64       `bson:"revalidates"` // Unix epoch seconds when this document should be removed from the cache.

	journal.Journal `bson:"-,inline"`
}

func NewValue() Value {
	return Value{
		ValueID:    primitive.NewObjectID(),
		URLs:       make([]string, 0, 1),
		Object:     make(mapof.Any),
		HTTPHeader: make(http.Header),
		Metadata:   streams.NewMetadata(),
	}
}

func (value Value) ID() string {
	return value.ValueID.Hex()
}

func (value Value) DocumentID() string {
	if objectID := value.Object.GetString(vocab.PropertyID); objectID != "" {
		return objectID
	}
	return value.URLs.First()
}

func (value Value) AsDocument() streams.Document {
	return streams.NewDocument(
		value.Object,
		streams.WithHTTPHeader(value.HTTPHeader),
		streams.WithMetadata(value.Metadata),
	)
}

// appendURL (safely) adds a URL to the value's list of URLs, avoiding duplicates and empty strings.
func (value *Value) AppendURL(url string) {

	if url == "" {
		return
	}

	if slices.Contains(value.URLs, url) {
		return
	}

	value.URLs = append(value.URLs, url)
}

// ShouldRevalidate returns TRUE if the "RevalidatesDate" is in the past.
func (value Value) ShouldRevalidate() bool {
	return (value.Revalidates > 0) && (value.Revalidates < time.Now().Unix())
}

// calcPublished calculates the date that a document was sent/refreshed by the origin.
// This IS NOT the original create or publish date.
func (value *Value) calcPublished() {

	if published := value.Object.GetTime(vocab.PropertyPublished); !published.IsZero() {
		value.Published = published.Unix()
		return
	}

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

	// Fall back to caching the document for 1 week
	value.Expires = time.Now().AddDate(0, 0, 7).Unix()
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
