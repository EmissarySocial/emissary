package activitypub

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// IsMLSCreate returns TRUE for a Create activity whose object is an inline (map-valued) MLS
// message that is not public-addressed. These are the activities the inbox never drops: group
// ciphertext must reach storage even from blocked or muted senders, or the missed message breaks
// the conversation's epoch ratchet for everyone.
func IsMLSCreate(document streams.Document) bool {

	// Only Create qualifies: every MLS message arrives as a Create, so all other types keep the
	// normal drop rules and the filter stays simple.
	if document.Type() != vocab.ActivityTypeCreate {
		return false
	}

	// RULE: the object must be INLINE. A link-shaped object's mediaType is unknowable without a
	// fetch this gate refuses to make, so it never qualifies. (Getters never fetch on their own;
	// the guard states the intent explicitly.)
	object := document.Object()

	if !object.IsMap() {
		return false
	}

	// The object must be MLS ciphertext
	if object.MediaType() != vocab.MediaTypeMLS {
		return false
	}

	// RULE: public-addressed "MLS" keeps the normal drop rules. Real group ciphertext is never
	// public, and this bounds how much junk a blocked sender can force into storage.
	if document.IsPublic() {
		return false
	}

	// Inline, private, ciphertext: the real deal.
	return true
}
