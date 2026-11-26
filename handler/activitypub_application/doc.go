// Package activitypub_application receives ActivityPub activities sent to this server's
// @application actor.
//
// The @application actor follows #hashtag relays (such as tags.pub) for the whole server.
// This inbox verifies each request, then routes it:
//   - Accept and Reject answer the actor's Follow of a relay
//   - Announce from a followed relay hands the post to the tag-follow delivery
//
// Anything else is accepted and ignored, as the inbox did before it handled relays.
package activitypub_application
