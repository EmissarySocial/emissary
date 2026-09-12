// Package ascontextmaker gives every ActivityPub document a conversation context.
//
// It is a hannibal streams.Client decorator.  As documents load, it walks the inReplyTo
// chain to the root of the conversation and stamps the result onto each document's "context"
// property.  A document with neither a context nor an inReplyTo is the root of its own
// conversation, and is marked as such.
//
// Context is what lets Emissary group a thread that arrived out of order, or in pieces from
// several servers, since a reply usually names its parent but rarely names the whole thread.
// Middleware order matters: place this decorator inside the cache so a context is computed
// once and reused, not recomputed on every load.
package ascontextmaker
