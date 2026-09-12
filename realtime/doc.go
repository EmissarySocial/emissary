// Package realtime pushes live updates to connected browsers.
//
// The Broker is a per-domain singleton that tracks every attached client and fans messages
// out to them, so a change made in one place shows up in open tabs with no page reload.
// Clients attach over Server-Sent Events and receive Messages addressed by topic.
//
// A Message names a stream and a topic (see the Topic constants), not a payload: the browser
// reacts by re-fetching the affected fragment over htmx.  Keeping the payload out of the
// message means a subscriber can never be handed data it is not authorized to see, since
// the re-fetch runs through the normal, authorized route.
package realtime
