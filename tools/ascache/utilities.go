package ascache

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
)

// FromCache returns TRUE if this document was retrieved from the cache database
func FromCache(document streams.Document) bool {
	return document.HTTPHeader().Get(HeaderHannibalCache) != ""
}

// stripCacheHeaders removes Hannibal's cache provenance headers from a document.
func stripCacheHeaders(document streams.Document) {

	// These headers say "this answer came from OUR cache", so a remote server must never be able to
	// say it for us.  They arrive over the wire like any other header, and FromCache() cannot tell a
	// forged one from a stamped one -- so the only place the distinction still exists is here, at the
	// moment the document crosses in from the interweb.
	header := document.HTTPHeader()
	header.Del(HeaderHannibalCache)
	header.Del(HeaderHannibalCacheDate)
}

// timeoutContext returns a context.Context that cancels itself after the designated number of seconds.
func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}

// asValue converts a streams.Document into a cacheable Value, calculating its freshness metadata.
func asValue(document streams.Document) Value {

	result := NewValue()
	result.URLs = append(result.URLs, document.ID())
	result.Object = document.Map()
	result.HTTPHeader = document.HTTPHeader()
	result.Metadata = document.Metadata

	// Calculate datetime metadata
	result.Received = time.Now().Unix()
	cacheControl := cacheheader.Parse(result.HTTPHeader)
	result.calcPublished()
	result.calcExpires(cacheControl)
	result.calcRevalidates(cacheControl)

	return result
}
