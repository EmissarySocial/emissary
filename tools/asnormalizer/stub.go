package asnormalizer

import (
	"net/http"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// stub returns the placeholder for a document that must not be loaded again: an object holding only its id.
func stub(uri string) streams.Document {

	// RULE: The stub is an object, never a bare string, because every getter on a bare string loads it.
	// The header marks intent only; ascache refuses to store the stub because it has no type.
	header := make(http.Header)
	header.Set(cacheheader.HeaderCacheControl, cacheheader.DirectiveNoStore)

	return streams.NewDocument(
		map[string]any{vocab.PropertyID: uri},
		streams.WithHTTPHeader(header),
	)
}
