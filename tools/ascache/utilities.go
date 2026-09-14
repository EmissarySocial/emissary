package ascache

import (
	"context"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/uri"
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

// timeoutContext returns a context.Context that cancels itself after the designated duration.
func timeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// sameHost returns TRUE if a lookup key and a document id are addressed to the same host
func sameHost(key string, documentID string) bool {

	host := lookupKeyHost(key)

	// A key with no host vouches for nothing
	if host == "" {
		return false
	}

	return host == lookupKeyHost(documentID)
}

// lookupKeyHost returns the lowercased host a lookup key is addressed to: a URL's host, the part
// of a @user@host handle after its last "@", or empty for anything else (a #hashtag, a bare word)
func lookupKeyHost(key string) string {

	// A URL carries its host in the authority
	if strings.HasPrefix(key, "https://") || strings.HasPrefix(key, "http://") {
		return uri.Hostname(key)
	}

	// A handle carries its host after the last "@". The key is classified BEFORE calling into
	// uri, because NormalizeHost would happily return "#tag" for a hashtag as if it were a host.
	if index := strings.LastIndex(key, "@"); index > 0 {
		return uri.NormalizeHost(key[index+1:])
	}

	return ""
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
