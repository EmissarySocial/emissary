package asnormalizer

import (
	"strconv"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/cespare/xxhash/v2"
)

// Stamp writes the metadata that is derived from a document's own contents: its category, its
// hashed ID, and the document it responds to. It leaves every other metadata field alone.
func Stamp(document *streams.Document) {

	document.Metadata.DocumentCategory = streams.DocumentCategory(document.Type())
	document.Metadata.HashedID = HashedID(document.ID())

	// RULE: Only an object records what it responds to
	if !document.IsObject() {
		return
	}

	// Response COUNTS are not computed here: ascache's CalcParentRelationships maintains them from
	// locally-cached evidence, and it is the single writer on purpose (two writers would fight).
	document.Metadata.RelationType, document.Metadata.RelationHref = calcRelationType(*document)
}

// HashedID returns the short identifier that names a document's elements on a page. It depends
// on the document's ID alone, so it can be computed for a document that was never stamped.
func HashedID(documentID string) string {
	return strconv.FormatUint(xxhash.Sum64String(documentID), 32)
}

// calcRelationType calculates the "RelationType" and "RelationHref" metadata for this
// cached document.
func calcRelationType(document streams.Document) (string, string) {

	// Get the document type

	// Calculate RelationType
	switch documentType := document.Type(); documentType {

	// Announce, Like, and Dislike are written straight to the cache.
	case vocab.ActivityTypeAnnounce,
		vocab.ActivityTypeLike,
		vocab.ActivityTypeDislike:

		return documentType, document.Object().ID()

	// Otherwise, see if this is a "Reply"
	default:
		unwrapped := document.UnwrapActivity()

		if inReplyTo := unwrapped.InReplyTo(); inReplyTo.NotNil() {
			return vocab.RelationTypeReply, inReplyTo.String()
		}
	}

	return "", ""
}
