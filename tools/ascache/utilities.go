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

func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}

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
