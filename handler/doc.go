// Package handler contains Emissary's HTTP handlers -- the functions wired to routes that turn an
// incoming request into a response.  Handlers here cover the web UI, the JSON API, file and
// attachment serving, OAuth, and sign-in.
//
// Handlers stay thin.  A handler parses its request, calls into the service and build packages,
// and writes the result; the rules themselves live in service.
//
// Most of that thinness comes from the WithXXX wrappers in wrappers.go.  Each one resolves a
// domain Factory, opens a database session (read-only for GET, a transaction for everything
// else), loads and authorizes the objects named in the URL, and only then calls the handler
// underneath it.  A handler that takes a *model.User has already had that User loaded and its
// visibility checked.
//
// Protocol-specific handlers live in subdirectories: activitypub_user, activitypub_stream,
// activitypub_domain, and activitypub_search for federation, mastodon for the
// Mastodon-compatible API, plus third-party integrations such as stripe and unsplash.
//
// Several routes serve one URL as either an HTML page or an ActivityStreams document, chosen by
// the request's Accept header.  That choice, and the response headers implied by it, belong to
// tools/headers -- so that the HEAD and GET handlers for a resource always describe it the
// same way.
package handler
