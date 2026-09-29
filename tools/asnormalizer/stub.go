package asnormalizer

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// stub returns the placeholder for a document that must not be loaded again: an object holding only its id.
func stub(uri string) streams.Document {

	// RULE: The stub is an object, never a bare string, because every getter on a bare string loads it.
	// It is marked NoStore, so that no cache ever serves it as the document for its URL.
	return streams.NewDocument(
		mapof.Any{vocab.PropertyID: uri},
		streams.WithNoStore(),
	)
}
