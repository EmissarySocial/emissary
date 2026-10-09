// Package asstrict provides strictly-typed views of ActivityStreams documents.
//
// Hannibal represents a document as a loose property tree, which is right for parsing
// whatever the network sends but awkward for code that knows exactly which fields it needs.
// The types here -- Activity, Actor, Object, Image, Tag, Attachment, PublicKey, and friends
// -- pull those fields into plain Go structs with plain Go types.
//
// The constructors are total: a missing or wrongly-typed property yields a zero value rather
// than an error, so a malformed remote document degrades into an empty field instead of
// failing a whole request.  Callers that must distinguish absent from empty should read the
// underlying document instead.
package asstrict
