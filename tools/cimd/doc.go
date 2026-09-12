// Package cimd reads Client ID Metadata Documents.
//
// A Client ID Metadata Document lets an OAuth client identify itself by URL instead of by
// pre-registration: the client id IS a URL, and fetching it returns the metadata a server
// needs to describe and authorize that client.  The specification lives at https://client.dev
//
// GetMetadata fetches and parses one.  The client id is attacker-supplied by nature, so
// callers must treat a returned Metadata as a claim the client makes about itself, never as
// something the server has verified.
package cimd
