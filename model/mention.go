package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// Mention represents a single @mention found in a document's content, paired with the
// ActivityPub Actor URL that its handle resolves to.
type Mention struct {
	// Handle and Href are stored TOGETHER so that extraction -- which re-runs on every save --
	// can recognize a handle it has already resolved and keep its Href.  Storing handles alone
	// would spend a WebFinger round trip per handle on every save of the document.
	Handle string `bson:"handle,omitempty"` // WebFinger handle, WITHOUT the "@" prefix (e.g. "user@server.social")

	// Empty until resolved at publish time.  An unresolved Mention is retained (the author may
	// fix the handle and re-publish) but is never federated: a `Mention` tag with no `href`
	// gives a receiving server nothing to route to.
	Href string `bson:"href,omitempty"` // ActivityPub Actor URL. Empty until resolved.
}

// NewMention returns a fully initialized (but unresolved) Mention for the provided handle.
func NewMention(handle string) Mention {
	return Mention{
		Handle: handle,
	}
}

// NewMentions returns a fully initialized slice of Mentions
func NewMentions() sliceof.Object[Mention] {
	return make(sliceof.Object[Mention], 0)
}

// IsResolved returns TRUE if this Mention has been resolved to an Actor URL.
func (mention Mention) IsResolved() bool {
	return mention.Href != ""
}

// NotResolved returns TRUE if this Mention has NOT yet been resolved to an Actor URL.
func (mention Mention) NotResolved() bool {
	return !mention.IsResolved()
}

// Name returns the display name for this Mention, in "@user@server.social" form.
func (mention Mention) Name() string {

	// This is published as the `name` of the AS2 Mention tag, and mirrors the microsyntax
	// that appears in the document's content.
	return "@" + mention.Handle
}

// JSONLD returns a JSON-LD map document representing this Mention as an AS2 `Mention`
// link (AS2 Vocabulary 5.6).
func (mention Mention) JSONLD() mapof.String {

	// NOTE: Callers must only publish RESOLVED mentions.  An empty `href` is not routable.
	return mapof.String{
		vocab.PropertyType: vocab.LinkTypeMention,
		vocab.PropertyName: mention.Name(),
		vocab.PropertyHref: mention.Href,
	}
}
