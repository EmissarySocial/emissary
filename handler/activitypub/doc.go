// Package activitypub holds the ActivityPub pieces that every federation handler shares.  The four
// protocol-specific packages -- activitypub_user, activitypub_stream, activitypub_domain, and
// activitypub_search -- import it so that an inbound activity is received the same way and an
// outbound collection is shaped the same way, whichever actor owns the URL.
//
// Inbound, ReceiveRequest is the one funnel.  It installs the canonical validator chain, verifies
// the HTTP signature, and strips reserved "emissary:" properties before the activity reaches any
// handler.  Because all four inbox families pass through it, the chain and the sanitizer cannot
// drift apart, and a security fix lands in one place rather than four.
//
// The chain's first link is RuleValidator, Stage 1 of the inbox block gate.  It reads the CLAIMED
// (still unverified) actor and the signature keyId, so it may DENY but never GRANT -- returning
// ResultValid would short-circuit the chain and skip signature verification entirely.  Stage 2, the
// authoritative gate on the VERIFIED actor, runs after verification, in the inbox handlers
// themselves.  IsWireGateException and IsMLSCreate name the activities Stage 1 must never discard.
//
// Outbound, Collection and CollectionPage build the paged OrderedCollections that back the outbox,
// followers, liked, featured, and children URLs.  Serving them from here is what keeps every
// collection's paging shape identical across actor types.
package activitypub
