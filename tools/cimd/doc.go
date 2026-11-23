// Package cimd reads Client ID Metadata Documents.
//
// A Client ID Metadata Document lets an OAuth client identify itself by URL instead of by
// pre-registration: the client id IS a URL, and fetching it returns the metadata a server
// needs to describe and authorize that client.  The specification lives at https://client.dev
//
// ParseClientID checks a URL client id against the IndieAuth rules and returns its canonical
// form.  GetMetadata fetches the document, and reports separately when a client answers but
// publishes none (an IndieAuth client serving an HTML page), so the caller can describe that
// client from its URL alone.  Only a 2xx answer names a client, and a document must be served
// at the client id itself: a redirect is followed to a page, and never to a document.  The client id is attacker-supplied by nature, so callers must
// treat a returned Metadata as a claim the client makes about itself, never as something the
// server has verified.
package cimd
