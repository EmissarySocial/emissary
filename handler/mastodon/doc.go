// Package mastodon implements Emissary's Mastodon-compatible client API.
//
// Each handler is a closure over a *server.Factory that returns the typed request/response
// function benpate/toot expects; benpate/toot-echo registers those on the echo router and
// enforces the OAuth scope attached to each route.
//
// Scope is not authorization.  toot-echo checks only that the Bearer token carries the right
// scope, which any valid token for that scope satisfies -- it does not check whether the
// caller may touch the specific object named in the request.  Every handler that loads a
// user-owned record must authorize that record itself, or the route is an IDOR.
//
// See AGENTS.md for that rule in full, plus the object-mapping and pagination conventions.
package mastodon
